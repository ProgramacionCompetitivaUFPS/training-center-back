package handler

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err *apperror.AppError) {
	status := err.StatusCode
	if status == 0 {
		status = http.StatusInternalServerError
	}
	
	writeJSON(w, status, err)
}
