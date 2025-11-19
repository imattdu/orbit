package httpclient

import (
	"context"
	"net"
	"net/http"
	"time"
)

// 构造 http.Transport
func buildTransport(cfg *Config) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: makeDialContext(
			cfg.DialTimeout,
			cfg.DialKeepAlive,
			cfg.ReadWriteTimeout,
		),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
	}
}

// timeoutConn 在每次 Read/Write 前设置 deadline，控制每次读写超时
type timeoutConn struct {
	net.Conn
	rw time.Duration
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	if c.rw > 0 {
		_ = c.SetReadDeadline(time.Now().Add(c.rw))
	}
	return c.Conn.Read(b)
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	if c.rw > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(c.rw))
	}
	return c.Conn.Write(b)
}

// 包装 DialContext，增加读写超时
func makeDialContext(dial, keepAlive, rw time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: dial, KeepAlive: keepAlive}
	if rw <= 0 {
		return d.DialContext
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &timeoutConn{Conn: conn, rw: rw}, nil
	}
}
