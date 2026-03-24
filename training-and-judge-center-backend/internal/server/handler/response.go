package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		slog.Error("unexpected internal error in handler", "error", err)
		appErr = apperror.NewInternal()
	}

	status := appErr.StatusCode
	if status == 0 {
		status = http.StatusInternalServerError
	}

	WriteJSON(w, status, appErr)
}
