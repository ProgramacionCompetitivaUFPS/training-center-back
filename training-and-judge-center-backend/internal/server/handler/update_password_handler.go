package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	// Rate limit: 5 attempts per hour
	rateKey := "rate_limit:update_password:" + claims.UserID
	allowed, err := h.rateLimiter.Allow(r.Context(), rateKey, 5, 1*time.Hour)
	if err != nil {
		// Log error but continue? Or fail safe? Usually fail safe is better for security, but here we'll log and continue if it's a Redis error to avoid blocking the user.
		// However, a better approach is to fail if we can't verify the rate limit.
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "INTERNAL_ERROR",
			"message": "An internal error occurred while processing your request",
		})
		return
	}
	if !allowed {
		respondJSON(w, http.StatusTooManyRequests, map[string]string{
			"error":   "TOO_MANY_REQUESTS",
			"message": "Too many password update attempts. Please try again in an hour.",
		})
		return
	}

	if err := h.updatePassword.Execute(r.Context(), claims.UserID, appuser.UpdatePasswordInput{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		respondError(w, err)
		return
	}

	// Reset rate limit on success
	_ = h.rateLimiter.Reset(r.Context(), rateKey)

	w.WriteHeader(http.StatusNoContent)
}
