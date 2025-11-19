package tracex

import (
	"context"
	"net/http"
	"time"
)

const (
	HeaderTraceID      = "X-Trace-Id"
	HeaderSpanID       = "X-Span-Id"
	HeaderParentSpanID = "X-Parent-Span-Id"
)

// -------------------- HTTP 头注入 / 提取 --------------------

// InjectToHeader 把当前 span 的 trace 信息注入 HTTP 头
func InjectToHeader(ctx context.Context, h http.Header) {
	if h == nil {
		return
	}
	span := SpanFromContext(ctx)
	if span == nil {
		return
	}
	if span.TraceID != "" {
		h.Set(HeaderTraceID, span.TraceID)
	}
	if span.SpanID != "" {
		h.Set(HeaderSpanID, span.SpanID)
	}
	if span.Parent != "" {
		h.Set(HeaderParentSpanID, span.Parent)
	}
}

// ExtractRemoteSpan 从 HTTP 头解析“远端 span 信息”（通常用于 server 端），
// 返回 remoteSpan（对方传来的）以及新的本地 ctx 和本地 span：
//
// 语义：
//
//	remoteSpan = 上游传来的 span（如果 header 中有）
//	localSpan  = 以 remoteSpan 作为 parent（或以 traceID 为根）创建的新 span
func ExtractRemoteSpan(ctx context.Context, h http.Header, name string) (context.Context, *Span, *Span) {
	var (
		traceID = ""
		spanID  = ""
		parent  = ""
	)

	if h != nil {
		traceID = h.Get(HeaderTraceID)
		spanID = h.Get(HeaderSpanID)
		parent = h.Get(HeaderParentSpanID)
	}

	var remote *Span
	if traceID != "" || spanID != "" {
		remote = &Span{
			TraceID: traceID,
			SpanID:  spanID,
			Parent:  parent,
		}
	}

	// 本地 span：以 remoteSpan 为 parent
	if traceID == "" {
		traceID = newID()
	}
	local := &Span{
		TraceID: traceID,
		SpanID:  newID(),
		Start:   time.Now(),
		Name:    name,
	}
	if spanID != "" {
		local.Parent = spanID
	} else if parent != "" {
		local.Parent = parent
	}

	ctx = WithSpan(ctx, local)
	return ctx, local, remote
}
