package httpclient

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Hook 在请求前后执行

type BeforeFunc func(ctx context.Context, req *http.Request)
type AfterFunc func(ctx context.Context, req *http.Request, resp *http.Response, err error)

// Config 是 Client 的初始化配置
type Config struct {
	BaseURL string

	// 请求级默认超时（per-request 没设 Timeout 时使用）
	DefaultTimeout time.Duration

	// 连接相关
	DialTimeout           time.Duration
	DialKeepAlive         time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	ReadWriteTimeout      time.Duration // 每次 Read/Write 的 deadline

	// 重试相关
	RetryMaxAttempts int
	RetryDecider     RetryDecider
	RetryBackoff     BackoffFunc

	// 业务错误解析
	BizErrDecoder BizErrorDecoder

	// Hook
	Before []BeforeFunc
	After  []AfterFunc

	// 调用统计上报（例如打日志）
	StatsHook StatsHook
}

func defaultConfig() Config {
	return Config{
		DefaultTimeout:        5 * time.Second,
		DialTimeout:           3 * time.Second,
		DialKeepAlive:         60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ReadWriteTimeout:      5 * time.Second,
		RetryMaxAttempts:      1,
	}
}

type Option func(*Config)

func WithBaseURL(s string) Option {
	return func(c *Config) { c.BaseURL = s }
}

func WithDefaultTimeout(t time.Duration) Option {
	return func(c *Config) { c.DefaultTimeout = t }
}

func WithReadWriteTimeout(t time.Duration) Option {
	return func(c *Config) { c.ReadWriteTimeout = t }
}

func WithBeforeHooks(h ...BeforeFunc) Option {
	return func(c *Config) { c.Before = append(c.Before, h...) }
}

func WithAfterHooks(h ...AfterFunc) Option {
	return func(c *Config) { c.After = append(c.After, h...) }
}

func WithRetry(max int, decider RetryDecider, backoff BackoffFunc) Option {
	return func(c *Config) {
		c.RetryMaxAttempts = max
		c.RetryDecider = decider
		c.RetryBackoff = backoff
	}
}

func WithBizErrorDecoder(dec BizErrorDecoder) Option {
	return func(c *Config) { c.BizErrDecoder = dec }
}

func WithStatsHook(h StatsHook) Option {
	return func(c *Config) { c.StatsHook = h }
}

// Client 是并发安全的 HTTP 客户端
type Client struct {
	hc      *http.Client
	baseURL *url.URL

	before []BeforeFunc
	after  []AfterFunc

	defaultTimeout   time.Duration
	retryMaxAttempts int
	retryDecider     RetryDecider
	backoff          BackoffFunc
	bizErrDecoder    BizErrorDecoder
	statsHook        StatsHook
}

// New 创建 Client，Config 初始化后不再修改 → 并发安全
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var base *url.URL
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return nil, err
		}
		base = u
	}

	tr := buildTransport(&cfg)

	maxAttempts := cfg.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	dec := cfg.RetryDecider
	if dec == nil {
		dec = defaultRetryDecider
	}
	bf := cfg.RetryBackoff
	if bf == nil {
		bf = defaultBackoff
	}

	return &Client{
		hc:      &http.Client{Transport: tr},
		baseURL: base,

		before: append([]BeforeFunc(nil), cfg.Before...),
		after:  append([]AfterFunc(nil), cfg.After...),

		defaultTimeout:   cfg.DefaultTimeout,
		retryMaxAttempts: maxAttempts,
		retryDecider:     dec,
		backoff:          bf,
		bizErrDecoder:    cfg.BizErrDecoder,
		statsHook:        cfg.StatsHook,
	}, nil
}
