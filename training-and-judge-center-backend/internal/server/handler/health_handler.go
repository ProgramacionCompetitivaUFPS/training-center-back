package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// @Summary      Health check
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      500 {object} apperror.AppError
// @Router       /ping [get]
func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
}
