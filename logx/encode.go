package logx

import (
	"context"
	"log/slog"

	"github.com/imattdu/orbit/cctx"
	"github.com/imattdu/orbit/errorx"
	"github.com/imattdu/orbit/tracex"
)

// encodeLog 把 ctx / tag / msg / kv 整合成一组 slog.Attr
func encodeLog(ctx context.Context, tag string, msg any, kv ...any) []slog.Attr {
	attrs := make([]slog.Attr, 0, 16)

	// tag
	if tag != "" {
		attrs = append(attrs, slog.String("tag", tag))
	}

	// caller
	c := getCaller()
	attrs = append(attrs,
		slog.String("file", c.file),
		slog.Int("line", c.line),
		slog.String("func", c.funcName),
	)

	// trace（来自 tracex）
	if span := tracex.SpanFromContext(ctx); span != nil {
		attrs = append(attrs,
			slog.String("trace_id", span.TraceID),
			slog.String("span_id", span.SpanID),
		)
	}

	// errorx 集成（如果 msg 是 *errorx.Error 或 error）
	switch v := msg.(type) {
	case *errorx.Error:
		attrs = append(attrs,
			slog.Int("code", v.Code.Code),
			slog.String("code_msg", v.Code.Message),
			slog.String("err_type", v.Type.Message),
			slog.String("service", v.Service.Message),
			slog.Bool("success", v.Success),
		)
		for k, vv := range v.Fields {
			attrs = append(attrs, slog.Any(k, vv))
		}
		// 再单独留一个 msg 字段
		if v.Message != "" {
			attrs = append(attrs, slog.String("msg", v.Message))
		}
	case error:
		attrs = append(attrs, slog.String("error", v.Error()))
	default:
		attrs = append(attrs, slog.Any("msg", v))
	}

	// cctx 中的通用字段（比如 biz_tag / env / caller_sys 等）
	if bag := cctx.All(ctx); len(bag) > 0 {
		for k, v := range bag {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	// 额外 kv（必须是偶数个）
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(k, kv[i+1]))
	}

	return attrs
}
