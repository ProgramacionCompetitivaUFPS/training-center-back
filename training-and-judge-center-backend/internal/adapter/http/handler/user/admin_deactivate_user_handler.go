package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Deactivate user (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users/{id}/deactivate [post]
func (h *Handler) AdminDeactivateUser(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	targetID := chi.URLParam(r, "id")

	if err := h.adminDeactivateUser.Execute(r.Context(), appuser.AdminDeactivateUserInput{
		RequesterID: cu.ID,
		TargetID:    targetID,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
