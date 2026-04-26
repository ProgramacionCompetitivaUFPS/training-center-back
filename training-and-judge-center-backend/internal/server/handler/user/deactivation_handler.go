package user

import (
	"encoding/json"
	"errors"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

func (h *UserHandler) RequestDeactivation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	if err := h.requestDeactivation.Execute(ctx, appuser.RequestDeactivationInput{UserID: userID}); err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "A confirmation code has been sent to your email",
	})
}

type confirmDeactivationBody struct {
	Code string `json:"code"`
}

func (h *UserHandler) ConfirmDeactivation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	var body confirmDeactivationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if !digitCodeRegex.MatchString(body.Code) {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "code", "message": "Code must be exactly 6 digits"},
			},
		})
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	input := appuser.ConfirmDeactivationInput{
		UserID:    userID,
		Code:      body.Code,
		IP:        &ip,
		UserAgent: &userAgent,
	}

	err := h.confirmDeactivation.Execute(ctx, input)
	if err != nil && !errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteError(w, err)
		return
	}

	if errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your account has been deactivated. We couldn't immediately close all active sessions — they will expire naturally.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
