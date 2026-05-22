package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type resetPasswordBody struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// @Summary      Reset password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body resetPasswordBody true "Reset data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} apperror.AppError
// @Router       /password/reset [post]
func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body resetPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" || !digitCodeRegex.MatchString(body.Code) || body.NewPassword == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]interface{}{
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

	out, err := h.resetPassword.Execute(ctx, input)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if !out.SessionsInvalidated {
		handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your password was reset successfully. We couldn't close your other active sessions — to close them, please change your password again.",
		})
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully. Please log in with your new password",
	})
}
