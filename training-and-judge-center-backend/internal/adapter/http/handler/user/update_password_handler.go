package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type updatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// @Summary      Update password
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body updatePasswordRequest true "Password data"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users/password [put]
func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	out, err := h.updatePassword.Execute(r.Context(), appuser.UpdatePasswordInput{
		UserID:          cu.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if !out.SessionsInvalidated {
		handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your password was changed successfully. We couldn't close your other active sessions — to close them, please change your password again.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
