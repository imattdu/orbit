package errorx

import (
	"errors"
	"fmt"
)

// Error 是统一错误类型：带 code、type、service、message、cause、扩展字段。
type Error struct {
	Code    CodeEntry      `json:"code"`    // 业务错误码
	Type    CodeEntry      `json:"type"`    // 错误类型：ErrTypeSys / ErrTypeBiz
	Success bool           `json:"success"` // 错误类型：ErrTypeSys / ErrTypeBiz
	Service CodeEntry      `json:"service"` // 出错模块：mysql / redis / service-X
	Message string         `json:"message"` // 用于覆盖 CodeEntry.Message
	Cause   error          `json:"-"`
	Fields  map[string]any `json:"fields,omitempty"`
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := e.Message
	if msg == "" {
		msg = e.Code.Message
	}
	if e.Cause != nil {
		return fmt.Sprintf("code=%d msg=%s cause=%v", e.Code.Code, msg, e.Cause)
	}
	return fmt.Sprintf("code=%d msg=%s", e.Code.Code, msg)
}

func (e *Error) Unwrap() error { return e.Cause }

// -------------------- Option --------------------

type Option func(*Error)

func WithMessage(msg string) Option {
	return func(e *Error) { e.Message = msg }
}

func WithCause(err error) Option {
	return func(e *Error) { e.Cause = err }
}

func WithType(t CodeEntry) Option {
	return func(e *Error) { e.Type = t }
}

func WithSuccess(success bool) Option {
	return func(e *Error) { e.Success = success }
}

func WithService(s CodeEntry) Option {
	return func(e *Error) { e.Service = s }
}

func WithField(k string, v any) Option {
	return func(e *Error) {
		if e.Fields == nil {
			e.Fields = make(map[string]any)
		}
		e.Fields[k] = v
	}
}

func WithFields(kv map[string]any) Option {
	return func(e *Error) {
		if len(kv) == 0 {
			return
		}
		if e.Fields == nil {
			e.Fields = make(map[string]any, len(kv))
		}
		for k, v := range kv {
			e.Fields[k] = v
		}
	}
}

// -------------------- 构造函数 --------------------

func New(code CodeEntry, opts ...Option) *Error {
	e := &Error{
		Code:    code,
		Type:    ErrTypeSys,
		Service: ServiceDefault,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func Newf(code CodeEntry, f string, args ...any) *Error {
	return New(code, WithMessage(fmt.Sprintf(f, args...)))
}

// NewBiz 业务错误
func NewBiz(code CodeEntry, opts ...Option) *Error {
	opts = append([]Option{WithType(ErrTypeBiz)}, opts...)
	return New(code, opts...)
}

// NewSys 系统错误
func NewSys(code CodeEntry, opts ...Option) *Error {
	opts = append([]Option{WithType(ErrTypeSys)}, opts...)
	return New(code, opts...)
}

// -------------------- Wrap --------------------

func Wrap(err error, code CodeEntry, opts ...Option) *Error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		// 已经是 Error：只补充 Option
		for _, opt := range opts {
			opt(e)
		}
		return e
	}

	opts = append([]Option{WithCause(err)}, opts...)
	return New(code, opts...)
}
