package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type requestEmailChangeBody struct {
	Password string `json:"password"`
	NewEmail string `json:"newEmail"`
}

// @Summary      Request email change
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body requestEmailChangeBody true "New email data"
// @Success      200 {object} map[string]string
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users/email-change/request [post]
func (h *UserHandler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
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

	var body requestEmailChangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Password == "" || body.NewEmail == "" {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
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

	output, err := h.requestEmailChange.Execute(ctx, input)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Verification code sent to the new email address",
		"expiresAt": output.ExpiresAt.Format(time.RFC3339),
	})
}

type confirmEmailChangeBody struct {
	Code string `json:"code"`
}

// @Summary      Confirm email change
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body confirmEmailChangeBody true "Confirmation code"
// @Success      200 {object} map[string]string
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users/email-change/confirm [post]
func (h *UserHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
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

	var body confirmEmailChangeBody
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

	input := appuser.ConfirmEmailChangeInput{
		UserID: userID,
		Code:   body.Code,
	}

	out, err := h.confirmEmailChange.Execute(ctx, input)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Email updated successfully",
		"email":   out.Email,
	})
}
