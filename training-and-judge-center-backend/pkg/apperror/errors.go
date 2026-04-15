package apperror

import (
	"errors"
	"net/http"
)

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

const (
	ErrCodeCannotSelfDeactivate  = "CANNOT_SELF_DEACTIVATE"
	ErrCodeCannotDeactivateAdmin = "CANNOT_DEACTIVATE_ADMIN"
)

func NewValidation(details []FieldError) *AppError {
	return &AppError{
		Code:       ErrCodeValidationError,
		Message:    "Invalid request data",
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

func NewBadRequest(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewConflict(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

func NewNotFound(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

func NewUnauthorized(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewForbidden(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

func NewInternal() *AppError {
	return &AppError{
		Code:       ErrCodeInternalError,
		Message:    "An unexpected error occurred",
		StatusCode: http.StatusInternalServerError,
	}
}

func NewServiceUnavailable(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusServiceUnavailable,
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
		Code:       code,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
		RetryAfter: retryAfter,
	}
}
