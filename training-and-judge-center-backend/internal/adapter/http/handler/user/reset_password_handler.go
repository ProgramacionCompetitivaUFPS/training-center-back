package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
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
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body resetPasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	if body.Email == "" || !digitCodeRegex.MatchString(body.Code) || body.NewPassword == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "invalid request data",
			Details: []apperror.FieldError{{Field: "email/code/newPassword", Message: "all fields are required"}},
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
