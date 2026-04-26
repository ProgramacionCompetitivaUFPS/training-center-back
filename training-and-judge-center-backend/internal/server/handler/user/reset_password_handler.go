package user

import (
	"encoding/json"
	"errors"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
)

type resetPasswordBody struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body resetPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" || !digitCodeRegex.MatchString(body.Code) || body.NewPassword == "" {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "email/code/newPassword", "message": "They are required"},
			},
		})
		return
	}

	input := appuser.ResetPasswordInput{
		Email:       body.Email,
		Code:        body.Code,
		NewPassword: body.NewPassword,
	}

	err := h.resetPassword.Execute(ctx, input)
	if err != nil && !errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteError(w, err)
		return
	}

	if errors.Is(err, appuser.ErrSessionsNotInvalidated) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your password was reset successfully. We couldn't close your other active sessions — to close them, please change your password again.",
		})
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully. Please log in with your new password",
	})
}
