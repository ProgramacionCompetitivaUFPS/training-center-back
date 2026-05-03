package apperror

import "encoding/json"

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type AppError struct {
	Code       string         `json:"error"`
	Message    string         `json:"message"`
	Details    []FieldError   `json:"details,omitempty"`
	RetryAfter int            `json:"retryAfter,omitempty"`
	Meta       map[string]any `json:"-"`
	StatusCode int            `json:"-"`
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

func (e *AppError) WithMeta(key string, value any) *AppError {
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}
	e.Meta[key] = value
	return e
}

func (e *AppError) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"error":   e.Code,
		"message": e.Message,
	}
	if len(e.Details) > 0 {
		m["details"] = e.Details
	}
	if e.RetryAfter != 0 {
		m["retryAfter"] = e.RetryAfter
	}
	for k, v := range e.Meta {
		m[k] = v
	}
	return json.Marshal(m)
}
