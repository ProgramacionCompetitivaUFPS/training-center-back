package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperror.AppError); ok {
		respondJSON(w, appErr.StatusCode, appErr)
		return
	}
	slog.Error("unhandled non-AppError reached respondError", "error", err)
	respondJSON(w, http.StatusInternalServerError, apperror.NewInternal())
}
