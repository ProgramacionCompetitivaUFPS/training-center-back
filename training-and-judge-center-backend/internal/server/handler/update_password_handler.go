package handler

import (
	"encoding/json"
	"log/slog"
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
		// Fail-Safe: If we can't verify the rate limit (e.g. Redis down), we block the request to prevent potential brute-force.
		slog.Error("failed to verify rate limit for update password", "user_id", claims.UserID, "error", err)
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

	err = h.updatePassword.Execute(r.Context(), appuser.UpdatePasswordInput{
		UserID:          claims.UserID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil && err != appuser.ErrSessionsNotInvalidated {
		respondError(w, err)
		return
	}

	// Reset rate limit on success
	_ = h.rateLimiter.Reset(r.Context(), rateKey)

	if err == appuser.ErrSessionsNotInvalidated {
		respondJSON(w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your password was changed successfully. We couldn't close your other active sessions — to close them, please change your password again.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
