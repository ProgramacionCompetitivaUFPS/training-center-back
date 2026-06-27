package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
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
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}
	userID := cu.ID
	ctx := r.Context()

	var body requestEmailChangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	if body.Password == "" || body.NewEmail == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "invalid request data",
			Details: []apperror.FieldError{{Field: "password/newEmail", Message: "all fields are required"}},
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
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]any{
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
func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}
	userID := cu.ID
	ctx := r.Context()

	var body confirmEmailChangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	if !digitCodeRegex.MatchString(body.Code) {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "invalid request data",
			Details: []apperror.FieldError{{Field: "code", Message: "code must be exactly 6 digits"}},
		})
		return
	}

	input := appuser.ConfirmEmailChangeInput{
		UserID: userID,
		Code:   body.Code,
	}

	out, err := h.confirmEmailChange.Execute(ctx, input)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]any{
		"message": "Email updated successfully",
		"email":   out.Email,
	})
}
