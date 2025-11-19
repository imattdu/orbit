package errorx

import "errors"

// From 提取 *Error
func From(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// -------------------- 类型判断 --------------------

func IsSuccess(err error) bool {
	e, ok := From(err)
	return ok && e.Success
}

func IsBiz(err error) bool {
	e, ok := From(err)
	return ok && e.Type.Code == ErrTypeBiz.Code
}

func IsSys(err error) bool {
	e, ok := From(err)
	return ok && e.Type.Code == ErrTypeSys.Code
}

// -------------------- 服务判断 --------------------

func ServiceOf(err error) CodeEntry {
	e, ok := From(err)
	if !ok {
		return ServiceDefault
	}
	return e.Service
}
