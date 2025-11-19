package tracex

import (
	"context"
	"time"
)

// Duration 返回 span 耗时
func (s *Span) Duration() time.Duration {
	if s == nil || s.Start.IsZero() || s.End.IsZero() {
		return 0
	}
	return s.End.Sub(s.Start)
}

type SpanHook func(ctx context.Context, span *Span)

var spanHook SpanHook

// SetGlobalSpanHook 设置全局 span 上报回调，例如：打日志、上报 trace 系统
func SetGlobalSpanHook(h SpanHook) {
	spanHook = h
}

// -------------------- Span 生命周期 --------------------

// StartSpan 在当前 ctx 上创建一个新的 span：
//   - 如果 ctx 中已有 span，则沿用 TraceID，并把当前 span 作为 parent
//   - 否则生成新的 TraceID，当前 span 为 root span
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	parent := SpanFromContext(ctx)

	traceID := ""
	parentID := ""
	if parent != nil {
		traceID = parent.TraceID
		parentID = parent.SpanID
	}
	if traceID == "" {
		traceID = newID()
	}

	span := &Span{
		TraceID: traceID,
		SpanID:  newID(),
		Parent:  parentID,
		Name:    name,
		Start:   time.Now(),
	}

	ctx = WithSpan(ctx, span)
	return ctx, span
}

// EndSpan 结束当前 ctx 对应的 span，并触发全局 Hook（如果有）
func EndSpan(ctx context.Context, err error) {
	span := SpanFromContext(ctx)
	if span == nil || !span.End.IsZero() {
		return
	}
	span.End = time.Now()
	if err != nil {
		span.Err = err
	}
	if spanHook != nil {
		spanHook(ctx, span)
	}
}

// EndSpanExplicit 结束指定 span，用于手里保存 *Span 的场景
func EndSpanExplicit(ctx context.Context, span *Span, err error) {
	if span == nil || !span.End.IsZero() {
		return
	}
	span.End = time.Now()
	if err != nil {
		span.Err = err
	}
	if spanHook != nil {
		spanHook(ctx, span)
	}
}
