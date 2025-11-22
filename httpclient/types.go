package httpclient

import (
	"context"
	"net/http"
	"time"
)

// CallAttempt 单次尝试信息
type CallAttempt struct {
	Attempt   int           `json:"attempt"`
	Status    int           `json:"status"`
	Err       error         `json:"err,omitempty"`
	Cost      time.Duration `json:"cost"`
	WillRetry bool          `json:"will_retry"`
	ctx       context.Context
}

// CallStats 一次完整调用信息
type CallStats struct {
	ctx context.Context
	// 请求级
	Method string `json:"method"`
	URL    string `json:"url"`
	Path   string `json:"path"`
	Query  string `json:"query"`

	// body 信息（可选）
	Body     string `json:"body,omitempty"`
	BodySize int    `json:"body_size,omitempty"`

	// 重试情况
	MaxAttempts int           `json:"max_attempts"`
	Attempts    int           `json:"attempts"`
	AttemptsLog []CallAttempt `json:"attempts_log,omitempty"`

	// 最终结果
	Status int           `json:"status"`
	Err    error         `json:"err,omitempty"`
	Cost   time.Duration `json:"cost"`

	Response interface{} `json:"response"`
}

// BizErrorDecoder 业务错误解析函数
type BizErrorDecoder func(statusCode int, body []byte) error

// StatsHook 统计上报 Hook（例如打日志）
type StatsHook func(ctx context.Context, stats *CallStats)

// ---------- 小工具 ----------

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	dst := make(http.Header, len(h))
	for k, vs := range h {
		cp := make([]string, len(vs))
		copy(cp, vs)
		dst[k] = cp
	}
	return dst
}

// []byte 深拷贝
func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	dst := make([]byte, len(b))
	copy(dst, b)
	return dst
}
