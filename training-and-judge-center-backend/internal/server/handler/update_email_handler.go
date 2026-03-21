package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type requestEmailChangeBody struct {
	Password string `json:"password"`
	NewEmail string `json:"newEmail"`
}

func (h *UserHandler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	// Rate Limiting: max 5 requests per hour for this user
	key := "email-change-request:" + userID
	allowed, err := h.rateLimiter.Allow(ctx, key, 5, time.Hour)
	if err != nil {
		respondError(w, apperror.NewInternal())
		return
	}
	if !allowed {
		respondJSON(w, http.StatusTooManyRequests, map[string]string{
			"error":   "RATE_LIMIT_EXCEEDED",
			"message": "Too many requests. Please try again later.",
		})
		return
	}

	var body requestEmailChangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Password == "" || body.NewEmail == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "password/newEmail", "message": "They are required"},
			},
		})
		return
	}

	input := appuser.RequestEmailChangeInput{
		UserID:   userID,
		Password: body.Password,
		NewEmail: body.NewEmail,
	}

	if err := h.requestEmailChange.Execute(ctx, input); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Verification code sent to the new email address",
		"expiresAt": time.Now().Add(15 * time.Minute).Format(time.RFC3339),
	})
}

type confirmEmailChangeBody struct {
	Code string `json:"code"`
}

func (h *UserHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	var body confirmEmailChangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Code == "" || len(body.Code) != 6 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "code", "message": "Code must be 6 characters long"},
			},
		})
		return
	}

	input := appuser.ConfirmEmailChangeInput{
		UserID: userID,
		Code:   body.Code,
	}

	newEmail, err := h.confirmEmailChange.Execute(ctx, input)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Email updated successfully",
		"email":   newEmail.String(),
	})
}
