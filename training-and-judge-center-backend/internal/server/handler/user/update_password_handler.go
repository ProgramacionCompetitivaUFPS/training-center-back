package user

import (
	"encoding/json"
	"errors"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	err := h.updatePassword.Execute(r.Context(), appuser.UpdatePasswordInput{
		UserID:          claims.UserID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil && !errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteError(w, err)
		return
	}

	if errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your password was changed successfully. We couldn't close your other active sessions — to close them, please change your password again.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
