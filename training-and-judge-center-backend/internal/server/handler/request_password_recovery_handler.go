package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type requestPasswordRecoveryBody struct {
	Email string `json:"email"`
}

func (h *UserHandler) RequestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body requestPasswordRecoveryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "email", "message": "Email is required"},
			},
		})
		return
	}

	// Rate Limiting
	key := "password-recovery:" + body.Email
	allowed, err := h.rateLimiter.Allow(ctx, key, 5, time.Hour)
	if err != nil {
		respondError(w, apperror.NewInternal())
		return
	}
	if !allowed {
		respondJSON(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":      "RATE_LIMIT_EXCEEDED",
			"message":    "Too many recovery requests. Please try again later",
			"retryAfter": 3600,
		})
		return
	}

	if err := h.requestPasswordRecovery.Execute(ctx, body.Email); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If the email is registered, a recovery code has been sent",
	})
}
