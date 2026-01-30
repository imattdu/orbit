package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/imattdu/orbit/errorx"
	"github.com/imattdu/orbit/logx"
	"github.com/imattdu/orbit/tracex"
)

// Do 发起请求：带重试、统计、业务错误解析
// respBody：
//   - nil       ：调用方自己处理 resp.Body（需自行 Close）
//   - io.Writer ：把响应体复制到 writer
//   - *[]byte   ：填充原始字节
//   - 其他      ：按 JSON 进行 Unmarshal
func (c *Client) Do(ctx context.Context, reqCfg *Request, respBody any) (*http.Response, error) {
	// ---------- 初始化统计 ----------
	stats := &CallStats{
		ctx:    ctx,
		Method: reqCfg.Method,
		Query:  reqCfg.Query.Encode(),
	}
	defer func() {
		logMap := map[string]interface{}{
			logx.Method:      stats.Method,
			logx.URL:         stats.URL,
			logx.Path:        stats.Path,
			logx.Query:       stats.Query,
			logx.Body:        stats.Body,
			"body_size":      stats.BodySize,
			logx.Attempts:    stats.Attempts,
			logx.MaxAttempts: stats.MaxAttempts,
			logx.Response:    stats.Response,
		}
		ctx := stats.ctx
		if stats.Attempts >= 1 {
			v := stats.AttemptsLog[stats.Attempts-1]
			ctx = v.ctx
			logMap[logx.Cost] = v.Cost / time.Millisecond
		}
		if stats.Err != nil {
			logMap[logx.Err] = stats.Err.Error()
		}

		if errorx.IsSuccess(stats.Err) {
			c.logger.Info(ctx, logx.TagHttpFailure, logMap)
		} else {
			c.logger.Warn(ctx, logx.TagHttpSuccess, logMap)
		}
	}()

	if ctx == nil {
		ctx = context.Background()
	}

	// ---------- per-request timeout ----------
	timeout := reqCfg.Timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}

	// ---------- URL ----------
	u, err := c.buildURL(reqCfg.Path, reqCfg.Query)
	if err != nil {
		return nil, err
	}
	stats.URL = u

	// ---------- Body 预处理（为了支持重试） ----------
	var (
		bodyBytes    []byte
		bodyReader   io.Reader
		bodyIsReader bool
	)
	headers := cloneHeader(reqCfg.Headers)

	switch v := reqCfg.Body.(type) {
	case nil:
	case io.Reader:
		bodyIsReader = true
		bodyReader = v
	default:
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(v); err != nil {
			stats.Err = err
			return nil, err
		}
		bodyBytes = cloneBytes(buf.Bytes())
		if headers == nil {
			headers = make(http.Header)
		}
		if headers.Get("Content-Type") == "" {
			headers.Set("Content-Type", "application/json")
		}
	}

	// ---------- 重试次数 ----------
	attempts := c.retryMaxAttempts
	if bodyIsReader {
		// io.Reader 不能重放，只能尝试一次
		attempts = 1
	}
	if attempts < 1 {
		attempts = 1
	}

	if bodyBytes != nil {
		stats.BodySize = len(bodyBytes)
		if len(bodyBytes) <= 1024 {
			stats.Body = string(bodyBytes)
		}
	}

	var lastResp *http.Response
	var lastErr error
	var isBreak bool
	begin := time.Now()
	stats.MaxAttempts = attempts
	// ---------- 重试主循环 ----------
	for attempt := 0; attempt < attempts; attempt++ {
		aAttempt := CallAttempt{
			Attempt: attempt + 1,
		}
		lastResp, isBreak, lastErr = func() (aResp *http.Response, isBreak bool, aErr error) {
			ctx, _ := tracex.StartSpan(ctx, "http")
			defer func() {
				tracex.EndSpan(ctx, nil)
				aAttempt.ctx = ctx
			}()
			ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
			defer timeoutCancel()

			// 每次重试重建 body reader
			if bodyBytes != nil {
				bodyReader = bytes.NewReader(bodyBytes)
			}
			httpReq, err := http.NewRequestWithContext(ctx, reqCfg.Method, u, bodyReader)
			if err != nil {
				return nil, true, err
			}
			for k, vs := range headers {
				for _, v := range vs {
					httpReq.Header.Add(k, v)
				}
			}

			if stats.Path == "" && httpReq.URL != nil {
				stats.Path = httpReq.URL.Path
			}
			if stats.Query == "" && httpReq.URL != nil {
				stats.Query = httpReq.URL.RawQuery
			}

			// before hook
			for _, h := range c.before {
				h(ctx, httpReq)
			}

			attemptStart := time.Now()
			resp, err := c.hc.Do(httpReq)
			aAttempt.Cost = time.Since(attemptStart)

			// after hook
			for _, h := range c.after {
				h(ctx, httpReq, resp, err)
			}

			if resp != nil {
				aAttempt.Status = resp.StatusCode
				if err == nil && resp.StatusCode != 200 {
					err = errorx.New(errorx.CodeEntry{
						Code:    resp.StatusCode,
						Message: resp.Status,
					})
				}
			}
			// 是否需要重试
			aAttempt.WillRetry = attempt < attempts-1 && c.retryDecider(resp, err)
			if !aAttempt.WillRetry {
				return resp, true, err
			}

			// 丢弃剩余 body，方便复用连接
			if resp != nil && resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			// 退避等待，支持 ctx 取消
			sleep := c.backoff(attempt)
			if sleep > 0 {
				select {
				case <-time.After(sleep):
				case <-ctx.Done():
					return nil, true, ctx.Err()
				}
			}
			return nil, false, err
		}()
		stats.AttemptsLog = append(stats.AttemptsLog, aAttempt)
		if isBreak {
			break
		}
	}

	// ---------- 填充最终统计 ----------
	stats.Cost = time.Since(begin)
	stats.Attempts = len(stats.AttemptsLog)
	stats.Err = lastErr

	// ---------- 整理返回 ----------
	if lastErr != nil && lastResp == nil {
		return nil, lastErr
	}
	resp := lastResp
	if resp == nil {
		return nil, lastErr
	}

	// 调用方自己处理 body
	if respBody == nil {
		return resp, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// io.Writer：流式复制
	if w, ok := respBody.(io.Writer); ok {
		_, err := io.Copy(w, resp.Body)
		stats.Err = err
		return resp, err
	}
	// 读完
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		stats.Err = err
		return resp, err
	}

	// 业务错误解析
	if c.bizErrDecoder != nil {
		if bErr := c.bizErrDecoder(resp.StatusCode, data); bErr != nil {
			stats.Err = bErr
			return resp, bErr
		}
	}

	// *[]byte：原始字节
	if p, ok := respBody.(*[]byte); ok {
		*p = data
		stats.Response = string(data)
		return resp, nil
	}
	// 默认 JSON
	if err := json.Unmarshal(data, respBody); err != nil {
		stats.Err = err
		return resp, err
	}
	stats.Response = respBody
	return resp, nil
}

// -------- 便捷方法 --------

func (c *Client) GetJSON(ctx context.Context, path string, out any, opts ...RequestOption) (*http.Response, error) {
	req := &Request{Method: http.MethodGet, Path: path}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req, out)
}

func (c *Client) PostJSON(ctx context.Context, path string, in any, out any, opts ...RequestOption) (*http.Response, error) {
	req := &Request{Method: http.MethodPost, Path: path}
	opts = append(opts, WithJSONBody(in))
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req, out)
}
