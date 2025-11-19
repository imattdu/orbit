package tracex

import (
	"context"

	"github.com/imattdu/orbit/cctx"
)

type spanKeyType struct{}

var spanKey = "span"

// SpanFromContext 取当前 ctx 中的 span
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	//v := ctx.Value(spanKey)
	span, _ := cctx.GetAs[*Span](ctx, spanKey)
	return span
}

// TraceIDFromContext 直接取 TraceID（没有则返回空串）
func TraceIDFromContext(ctx context.Context) string {
	if s := SpanFromContext(ctx); s != nil {
		return s.TraceID
	}
	return ""
}

// SpanNameFromContext 取 Span 名称
func SpanNameFromContext(ctx context.Context) string {
	if s := SpanFromContext(ctx); s != nil {
		return s.Name
	}
	return ""
}

// WithSpan 把 span 写入 ctx
func WithSpan(ctx context.Context, s *Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	//return context.WithValue(ctx, spanKey, s)
	return cctx.With(ctx, spanKey, s)
}
