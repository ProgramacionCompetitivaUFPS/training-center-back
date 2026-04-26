package problem

import (
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) ListModifiers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug is required"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	modifiers, err := h.listModifiersUC.Execute(r.Context(), appProblem.ListModifiersInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{"modifiers": modifiers})
}
