package problem

import (
	"context"
	"errors"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// 413 only when the body outgrew its cap; anything else is a malformed request.
func writeMultipartParseError(ctx context.Context, w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		handler.WriteJSON(ctx, w, http.StatusRequestEntityTooLarge, apperror.AppError{
			Code:    apperror.ErrCodePayloadTooLarge,
			Message: "File exceeds maximum allowed size",
		})
		return
	}
	handler.WriteError(ctx, w, apperror.NewValidation([]apperror.FieldError{
		{Field: "file", Message: "Request body must be multipart/form-data"},
	}))
}
