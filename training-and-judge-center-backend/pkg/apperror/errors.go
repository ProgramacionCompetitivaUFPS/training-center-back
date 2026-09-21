package apperror

import "errors"

func AccumulateFieldErrors(err error, fieldErrs *[]FieldError) error {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		*fieldErrs = append(*fieldErrs, appErr.Details...)
		return nil
	}
	return NewInternal()
}

func NewValidation(details []FieldError) *AppError {
	return &AppError{
		Kind:    KindValidation,
		Code:    ErrCodeValidationError,
		Message: "Invalid request data",
		Details: details,
	}
}

func NewBadRequest(code, message string) *AppError {
	return &AppError{
		Kind:    KindBadRequest,
		Code:    code,
		Message: message,
	}
}

func NewConflict(code, message string) *AppError {
	return &AppError{
		Kind:    KindConflict,
		Code:    code,
		Message: message,
	}
}

// NewConflictWithDetails is NewConflict plus field-level details, for a
// conflict the caller should highlight on a specific form field (e.g. a
// duplicate nickname) rather than show only as a generic message.
func NewConflictWithDetails(code, message string, details []FieldError) *AppError {
	return &AppError{
		Kind:    KindConflict,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewNotFound(code, message string) *AppError {
	return &AppError{
		Kind:    KindNotFound,
		Code:    code,
		Message: message,
	}
}

func NewUnauthorized(code, message string) *AppError {
	return &AppError{
		Kind:    KindUnauthorized,
		Code:    code,
		Message: message,
	}
}

func NewForbidden(code, message string) *AppError {
	return &AppError{
		Kind:    KindForbidden,
		Code:    code,
		Message: message,
	}
}

func NewInternal() *AppError {
	return &AppError{
		Kind:    KindInternal,
		Code:    ErrCodeInternalError,
		Message: "An unexpected error occurred",
	}
}

func NewServiceUnavailable(code, message string) *AppError {
	return &AppError{
		Kind:    KindServiceUnavailable,
		Code:    code,
		Message: message,
	}
}

func NewTooManyRequests(code, message string, retryAfter int) *AppError {
	if code == "" {
		code = "RATE_LIMIT_EXCEEDED"
	}
	if message == "" {
		message = "Too many requests. Please try again later"
	}
	return &AppError{
		Kind:       KindTooManyRequests,
		Code:       code,
		Message:    message,
		RetryAfter: retryAfter,
	}
}
