package apperror

import "net/http"

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
