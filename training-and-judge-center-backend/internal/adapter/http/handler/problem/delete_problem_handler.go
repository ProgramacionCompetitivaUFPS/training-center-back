package problem

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type deleteProblemRequest struct {
	ConfirmSlug string `json:"confirmSlug"`
}

// @Summary      Delete problem
// @Tags         problems
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        body body deleteProblemRequest true "Confirmation"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug} [delete]
func (h *Handler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var req deleteProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	err := h.deleteProblemUC.Execute(r.Context(), appProblem.DeleteProblemInput{
		Slug:        slug,
		ConfirmSlug: req.ConfirmSlug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
