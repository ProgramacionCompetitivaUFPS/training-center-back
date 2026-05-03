package apperror

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AppError struct {
	Code       string       `json:"error"`
	Message    string       `json:"message"`
	Details    []FieldError `json:"details,omitempty"`
	RetryAfter int          `json:"retryAfter,omitempty"`
	StatusCode int          `json:"-"`
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
