package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Remove problem modifier
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        nickname path string true "Nickname of the modifier"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/modifiers/{nickname} [delete]
func (h *Handler) RemoveModifier(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	nickname := r.PathValue("nickname")
	if slug == "" || nickname == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug and nickname are required"})
		return
	}

	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	err := h.removeModifier.Execute(r.Context(), appProblem.RemoveModifierInput{
		Slug:         slug,
		UserNickname: nickname,
		CurrentUser:  currentUser,
	})

	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
