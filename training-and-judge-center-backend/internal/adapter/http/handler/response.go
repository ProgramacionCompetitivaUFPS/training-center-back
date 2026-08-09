package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// WriteJSON and WriteError are exported for use by handler/problem subpackage.
func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON response", "error", err, "status", status)
	}
}

func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		slog.ErrorContext(ctx, "unexpected internal error in handler", "error", err)
		appErr = apperror.NewInternal()
	}
	WriteJSON(ctx, w, kindToStatus(appErr.Kind), appErr)
}

func kindToStatus(k apperror.Kind) int {
	switch k {
	case apperror.KindValidation:
		return http.StatusBadRequest
	case apperror.KindBadRequest:
		return http.StatusBadRequest
	case apperror.KindConflict:
		return http.StatusConflict
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindForbidden:
		return http.StatusForbidden
	case apperror.KindUnauthorized:
		return http.StatusUnauthorized
	case apperror.KindTooManyRequests:
		return http.StatusTooManyRequests
	case apperror.KindServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
