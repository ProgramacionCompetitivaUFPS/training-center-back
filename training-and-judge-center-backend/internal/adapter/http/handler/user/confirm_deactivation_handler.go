package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type confirmDeactivationBody struct {
	Code string `json:"code"`
}

// @Summary      Confirm account deactivation
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body confirmDeactivationBody true "Confirmation code"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users/deactivation/confirm [post]
func (h *Handler) ConfirmDeactivation(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}
	userID := cu.ID
	ctx := r.Context()

	var body confirmDeactivationBody
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

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	input := appuser.ConfirmDeactivationInput{
		UserID:    userID,
		Code:      body.Code,
		IP:        &ip,
		UserAgent: &userAgent,
	}

	out, err := h.confirmDeactivation.Execute(ctx, input)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if !out.SessionsInvalidated {
		handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
			"code":    "SESSIONS_NOT_INVALIDATED",
			"message": "Your account has been deactivated. We couldn't immediately close all active sessions — they will expire naturally.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
