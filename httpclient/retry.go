package httpclient

import (
	"net/http"
	"time"
)

// RetryDecider 决定某次响应是否需要重试
type RetryDecider func(resp *http.Response, err error) bool

// BackoffFunc 返回第 attempt 次重试前需要 sleep 的时间
type BackoffFunc func(attempt int) time.Duration

// 默认重试策略：网络错误 + 5xx
func defaultRetryDecider(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp != nil && resp.StatusCode >= 500 {
		return true
	}
	return false
}

// 默认指数退避：100ms, 200ms, 400ms, ... 最大 2s
func defaultBackoff(attempt int) time.Duration {
	base := 100 * time.Millisecond
	max := 2 * time.Second

	d := base << attempt
	if d > max {
		d = max
	}
	return d
}
