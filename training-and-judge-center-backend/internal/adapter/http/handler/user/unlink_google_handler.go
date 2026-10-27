package user

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Unlink Google account
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /users/google [delete]
func (h *Handler) UnlinkGoogle(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	if err := h.unlinkGoogle.Execute(r.Context(), appuser.UnlinkGoogleIdentityInput{UserID: cu.ID}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
