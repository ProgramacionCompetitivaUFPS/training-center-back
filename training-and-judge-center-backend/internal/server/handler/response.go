package handler

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperror.AppError); ok {
		respondJSON(w, appErr.StatusCode, appErr)
		return
	}
	respondJSON(w, http.StatusInternalServerError, apperror.NewInternal())
}
