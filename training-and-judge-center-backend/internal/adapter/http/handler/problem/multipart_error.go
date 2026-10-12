package problem

import (
	"errors"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// 413 only when the body outgrew its cap; anything else is a malformed request.
func multipartParseError(err error) *apperror.AppError {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return apperror.NewPayloadTooLarge("File exceeds maximum allowed size")
	}
	return apperror.NewValidation([]apperror.FieldError{
		{Field: "file", Message: "Request body must be multipart/form-data"},
	})
}
