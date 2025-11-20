package logx

import (
	"context"
	"log/slog"
	"time"
)

// Logger 对外暴露给业务 / 组件使用的接口
type Logger interface {
	Debug(ctx context.Context, tag string, msg any, kv ...any)
	Info(ctx context.Context, tag string, msg any, kv ...any)
	Warn(ctx context.Context, tag string, msg any, kv ...any)
	Error(ctx context.Context, tag string, msg any, kv ...any)
}

type loggerImpl struct {
	slog *slog.Logger
}

func (l *loggerImpl) Debug(ctx context.Context, tag string, msg any, kv ...any) {
	l.log(ctx, slog.LevelDebug, tag, msg, kv...)
}

func (l *loggerImpl) Info(ctx context.Context, tag string, msg any, kv ...any) {
	l.log(ctx, slog.LevelInfo, tag, msg, kv...)
}

func (l *loggerImpl) Warn(ctx context.Context, tag string, msg any, kv ...any) {
	l.log(ctx, slog.LevelWarn, tag, msg, kv...)
}

func (l *loggerImpl) Error(ctx context.Context, tag string, msg any, kv ...any) {
	l.log(ctx, slog.LevelError, tag, msg, kv...)
}

func (l *loggerImpl) log(ctx context.Context, level slog.Level, tag string, msg any, kv ...any) {
	if l == nil || l.slog == nil {
		return
	}
	if !l.slog.Enabled(ctx, level) {
		return
	}

	attrs := encodeLog(ctx, tag, msg, kv...)
	rec := slog.NewRecord(time.Now(), level, "", 0)
	rec.AddAttrs(attrs...)

	// 直接用 handler 处理（我们自定义的 handler 支持异步、切分等）
	_ = l.slog.Handler().Handle(ctx, rec)
}

// -------------------- 全局默认 logger --------------------

var defaultLogger Logger

// Init 根据 Config 初始化全局 logger（建议在 main 里调用一次）
func Init(cfg Config) error {
	h, err := newHandler(cfg)
	if err != nil {
		return err
	}
	defaultLogger = &loggerImpl{slog: slog.New(h)}
	return nil
}

// New 创建一个独立的 Logger 实例
func New(cfg Config) (Logger, error) {
	h, err := newHandler(cfg)
	if err != nil {
		return nil, err
	}
	return &loggerImpl{slog: slog.New(h)}, nil
}

// L 返回全局 logger，未 Init 时为 nil
func L() Logger {
	return defaultLogger
}

// 方便业务直接调用的快捷函数

func Debug(ctx context.Context, tag string, msg any, kv ...any) {
	if L() != nil {
		L().Debug(ctx, tag, msg, kv...)
	}
}

func Info(ctx context.Context, tag string, msg any, kv ...any) {
	if L() != nil {
		L().Info(ctx, tag, msg, kv...)
	}
}

func Warn(ctx context.Context, tag string, msg any, kv ...any) {
	if L() != nil {
		L().Warn(ctx, tag, msg, kv...)
	}
}

func Error(ctx context.Context, tag string, msg any, kv ...any) {
	if L() != nil {
		L().Error(ctx, tag, msg, kv...)
	}
}
