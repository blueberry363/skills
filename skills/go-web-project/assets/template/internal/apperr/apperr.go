// Package apperr 定义跨层共用的业务错误。
//
// service / model 层只返回这里的错误；HTTP 状态码与业务码的映射
// 集中在 internal/response，业务层不得 import gin 或 net/http。
package apperr

import "errors"

var (
	// ErrUnauthorized 未认证 → 401。
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden 已认证但无权 → 403。
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound 资源不存在 → 404。
	ErrNotFound = errors.New("not found")
	// ErrConflict 唯一约束冲突 → 409。
	ErrConflict = errors.New("conflict")
	// ErrInvalid 入参不合法 → 422。
	ErrInvalid = errors.New("invalid argument")
)

// InvalidError 携带字段名，便于前端定位。
type InvalidError struct {
	Field   string
	Message string
}

func (e *InvalidError) Error() string {
	return e.Field + ": " + e.Message
}

// Is 让 *InvalidError 能被 errors.Is(err, ErrInvalid) 命中。
func (e *InvalidError) Is(target error) bool {
	return target == ErrInvalid
}

// Invalid 构造字段级校验错误。
func Invalid(field, message string) error {
	return &InvalidError{Field: field, Message: message}
}
