package problem

import (
	"encoding/json"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Add problem modifier
// @Tags         problems
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        body body addModifierRequest true "User ID"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /problems/p/{slug}/modifiers [post]
func (h *Handler) AddModifier(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	var body struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body or missing userId"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	_, err := h.addModifierUC.Execute(r.Context(), appProblem.AddModifierInput{
		Slug:        slug,
		UserID:      body.UserID,
		CurrentUser: currentUser,
	})

	if err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
