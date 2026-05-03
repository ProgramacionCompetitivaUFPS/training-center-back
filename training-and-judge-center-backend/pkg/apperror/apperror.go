package apperror

type Kind string

const (
	KindValidation         Kind = "VALIDATION"
	KindConflict           Kind = "CONFLICT"
	KindNotFound           Kind = "NOT_FOUND"
	KindForbidden          Kind = "FORBIDDEN"
	KindUnauthorized       Kind = "UNAUTHORIZED"
	KindBadRequest         Kind = "BAD_REQUEST"
	KindInternal           Kind = "INTERNAL"
	KindTooManyRequests    Kind = "TOO_MANY_REQUESTS"
	KindServiceUnavailable Kind = "SERVICE_UNAVAILABLE"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AppError struct {
	Kind       Kind         `json:"-"`
	Code       string       `json:"error"`
	Message    string       `json:"message"`
	Details    []FieldError `json:"details,omitempty"`
	RetryAfter int          `json:"retryAfter,omitempty"`
	cause      error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.cause
}

func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}
