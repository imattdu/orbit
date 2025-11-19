package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Do 发起请求：带重试、统计、业务错误解析
// respBody：
//   - nil       ：调用方自己处理 resp.Body（需自行 Close）
//   - io.Writer ：把响应体复制到 writer
//   - *[]byte   ：填充原始字节
//   - 其他      ：按 JSON 进行 Unmarshal
func (c *Client) Do(ctx context.Context, reqCfg *Request, respBody any) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// ---------- per-request timeout ----------
	timeout := reqCfg.Timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// ---------- URL ----------
	u, err := c.buildURL(reqCfg.Path, reqCfg.Query)
	if err != nil {
		return nil, err
	}

	// ---------- Body 预处理（为了支持重试） ----------
	var bodyBytes []byte
	var bodyReader io.Reader
	var bodyIsReader bool

	headers := cloneHeader(reqCfg.Headers)

	switch v := reqCfg.Body.(type) {
	case nil:
	case io.Reader:
		bodyIsReader = true
		bodyReader = v
	default:
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(v); err != nil {
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

	// ---------- 初始化统计 ----------
	stats := &CallStats{
		Method:      reqCfg.Method,
		URL:         u,
		Query:       reqCfg.Query.Encode(),
		MaxAttempts: attempts,
	}
	if bodyBytes != nil {
		stats.BodySize = len(bodyBytes)
		if len(bodyBytes) <= 1024 {
			stats.Body = string(bodyBytes)
		}
	}

	var lastResp *http.Response
	var lastErr error
	begin := time.Now()

	// ---------- 重试主循环 ----------
	for attempt := 0; attempt < attempts; attempt++ {
		// 每次重试重建 body reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		httpReq, err := http.NewRequestWithContext(ctx, reqCfg.Method, u, bodyReader)
		if err != nil {
			return nil, err
		}
		for k, vs := range headers {
			for _, v := range vs {
				httpReq.Header.Add(k, v)
			}
		}

		if stats.Path == "" && httpReq.URL != nil {
			stats.Path = httpReq.URL.Path
		}

		// before hook
		for _, h := range c.before {
			h(ctx, httpReq)
		}

		attemptStart := time.Now()
		resp, err := c.hc.Do(httpReq)
		elapsed := time.Since(attemptStart)

		// after hook
		for _, h := range c.after {
			h(ctx, httpReq, resp, err)
		}

		lastResp, lastErr = resp, err

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		// 是否需要重试
		willRetry := attempt < attempts-1 && c.retryDecider(resp, err)

		// 记录单次尝试
		stats.AttemptsLog = append(stats.AttemptsLog, CallAttempt{
			Attempt:   attempt + 1,
			Status:    statusCode,
			Err:       errString(err),
			Cost:      elapsed,
			WillRetry: willRetry,
		})

		if !willRetry {
			break
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
				lastErr = ctx.Err()
				break
			}
		}
	}

	// ---------- 填充最终统计 ----------
	stats.Cost = time.Since(begin)
	stats.Attempts = len(stats.AttemptsLog)
	if lastResp != nil {
		stats.Status = lastResp.StatusCode
	}
	stats.Err = errString(lastErr)

	// 交给调用方打日志 / 上报
	if c.statsHook != nil {
		c.statsHook(ctx, stats)
	}

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
	defer resp.Body.Close()

	// io.Writer：流式复制
	if w, ok := respBody.(io.Writer); ok {
		_, err := io.Copy(w, resp.Body)
		return resp, err
	}

	// 读完
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	// 业务错误解析
	if c.bizErrDecoder != nil {
		if berr := c.bizErrDecoder(resp.StatusCode, data); berr != nil {
			return resp, berr
		}
	}

	// *[]byte：原始字节
	if p, ok := respBody.(*[]byte); ok {
		*p = data
		return resp, nil
	}

	// 默认 JSON
	if err := json.Unmarshal(data, respBody); err != nil {
		return resp, err
	}

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
