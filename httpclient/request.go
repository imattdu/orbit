package httpclient

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Request 表示一次请求的配置
type Request struct {
	Method  string
	Path    string      // 基于 BaseURL 的相对路径，或完整 URL
	Query   url.Values  // 额外 query
	Headers http.Header // 请求头
	Body    any         // nil / io.Reader / struct/map(会被 JSON 编码)

	Timeout time.Duration // per-request timeout（优先级高于 Config.DefaultTimeout）
}

type RequestOption func(*Request)

func WithQuery(q url.Values) RequestOption {
	return func(r *Request) { r.Query = q }
}

func WithHeader(k, v string) RequestOption {
	return func(r *Request) {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		r.Headers.Add(k, v)
	}
}

func WithJSONBody(body any) RequestOption {
	return func(r *Request) { r.Body = body }
}

func WithTimeout(t time.Duration) RequestOption {
	return func(r *Request) { r.Timeout = t }
}

func WithPathTemplate(format string, args ...any) RequestOption {
	return func(r *Request) { r.Path = fmt.Sprintf(format, args...) }
}

// buildURL 组合 baseURL + path + query
func (c *Client) buildURL(path string, q url.Values) (string, error) {
	// 1. path 是完整 URL
	if u, err := url.Parse(path); err == nil && u.Scheme != "" && u.Host != "" {
		qs := u.Query()
		for k, vs := range q {
			for _, v := range vs {
				qs.Add(k, v)
			}
		}
		u.RawQuery = qs.Encode()
		return u.String(), nil
	}

	// 2. 相对路径
	pu, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	// 没有 BaseURL：直接在相对路径上合并 query
	if c.baseURL == nil {
		qs := pu.Query()
		for k, vs := range q {
			for _, v := range vs {
				qs.Add(k, v)
			}
		}
		pu.RawQuery = qs.Encode()
		return pu.String(), nil
	}

	// 基于 BaseURL 拼接
	u := *c.baseURL
	u.Path = joinPath(c.baseURL.Path, pu.Path)

	qs := pu.Query()
	for k, vs := range q {
		for _, v := range vs {
			qs.Add(k, v)
		}
	}
	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// joinPath 简单处理一下 / 的拼接
func joinPath(a, b string) string {
	switch {
	case a == "" || a == "/":
		return b
	case b == "":
		return a
	default:
		if a[len(a)-1] == '/' && b[0] == '/' {
			return a + b[1:]
		}
		if a[len(a)-1] != '/' && b[0] != '/' {
			return a + "/" + b
		}
		return a + b
	}
}
