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
}

func (e *AppError) Error() string {
	return e.Message
}
