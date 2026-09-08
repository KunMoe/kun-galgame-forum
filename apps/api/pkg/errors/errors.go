package errors

import "fmt"

// One field the upstream named when it refused a write, carried through in the
// RFC 7807 shape the editing engine emits. Only the edit lane fills it; every
// other response omits the key entirely.
type FieldError struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
	Header    string `json:"header,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type AppError struct {
	Code       int          `json:"code"`
	Message    string       `json:"message"`
	StatusCode int          `json:"-"`
	Errors     []FieldError `json:"errors,omitempty"`
}

// WithFieldErrors returns a copy carrying the upstream's per-field messages, so
// a shared sentinel error value cannot be mutated by one request's rejection.
func (e *AppError) WithFieldErrors(errs []FieldError) *AppError {
	if len(errs) == 0 {
		return e
	}
	out := *e
	out.Errors = errs
	return &out
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func New(code int, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

const (
	CodeOK             = 0
	CodeAuth           = 205
	CodeBiz            = 233
	CodeBanned         = 234
	CodeReauthRequired = 235
)

func ErrUnauthorized(msg string) *AppError {
	return New(CodeAuth, msg, 401)
}

func ErrAuthExpired() *AppError {
	return New(CodeAuth, "用户登录失效", 401)
}

func ErrAccountBanned() *AppError {
	return New(CodeBanned, "账号已封禁", 403)
}

func ErrReauthRequired(msg string) *AppError {
	return New(CodeReauthRequired, msg, 403)
}

func ErrForbidden(msg string) *AppError {
	return New(CodeBiz, msg, 403)
}

func ErrBadRequest(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}

func ErrNotFound(msg string) *AppError {
	return New(CodeBiz, msg, 404)
}

func ErrInternal(msg string) *AppError {
	return New(CodeBiz, msg, 500)
}

func ErrValidation(msg string) *AppError {
	return New(CodeBiz, msg, 400)
}
