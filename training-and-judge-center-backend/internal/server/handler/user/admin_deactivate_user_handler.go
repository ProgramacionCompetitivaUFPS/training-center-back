package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
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
func (h *UserHandler) AdminDeactivateUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	targetID := chi.URLParam(r, "id")

	if err := h.adminDeactivateUser.Execute(r.Context(), appuser.AdminDeactivateUserInput{
		RequesterID: claims.UserID,
		TargetID:    targetID,
	}); err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
