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
	// The submission's title matches a live work. Catalog's gate is soft — the
	// same request with confirm_duplicates mints anyway — so the client has to
	// be able to tell this refusal apart from every other 409 and offer that
	// choice. Without its own code it read as a dead end in English.
	CodeDuplicateSuspects   = 236
	CodeIdempotencyInFlight = 237
	CodeIdempotencyMismatch = 238
	// Age attestation only ever happens at the account centre, so the client
	// has to tell this refusal apart from every other 403 to offer the
	// deep-link instead of a dead-end toast.
	CodeAdultConfirmationRequired = 239
	// The account has not granted this site the `preferences` scope. Every
	// cloud preference write will fail until it re-authorizes, so the client
	// degrades to cookies silently rather than toasting on each keystroke.
	CodeCloudPreferencesUnavailable = 240
	CodeCloudPreferencesConflict    = 241
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

func ErrDuplicateSuspects(msg string) *AppError {
	return New(CodeDuplicateSuspects, msg, 409)
}

func ErrAdultConfirmationRequired() *AppError {
	return New(CodeAdultConfirmationRequired, "请先在账号中心完成年龄确认", 403)
}

func ErrCloudPreferencesUnavailable() *AppError {
	return New(CodeCloudPreferencesUnavailable, "本站尚未获得云端偏好的写入授权", 403)
}

func ErrCloudPreferencesConflict() *AppError {
	return New(CodeCloudPreferencesConflict, "云端偏好已在别处更新, 请重新读取后再写", 412)
}
