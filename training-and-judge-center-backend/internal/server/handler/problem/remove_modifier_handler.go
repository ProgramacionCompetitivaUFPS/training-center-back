package problem

import (
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Remove problem modifier
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        userId path string true "User ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/modifiers/{userId} [delete]
func (h *Handler) RemoveModifier(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	userID := r.PathValue("userId")
	if slug == "" || userID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug and userId are required"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	_, err := h.removeModifierUC.Execute(r.Context(), appProblem.RemoveModifierInput{
		Slug:        slug,
		UserID:      userID,
		CurrentUser: currentUser,
	})

	if err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
