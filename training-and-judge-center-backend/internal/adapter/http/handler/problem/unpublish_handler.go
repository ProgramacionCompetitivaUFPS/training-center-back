package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type unpublishResponse struct {
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// @Summary      Unpublish problem
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} unpublishResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/unpublish [post]
func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	out, err := h.unpublishProblem.Execute(r.Context(), appProblem.UnpublishProblemInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, unpublishResponse{
		Slug:    out.Problem.Slug,
		Status:  out.Problem.Status,
		Message: "Problem unpublished successfully. You can now make changes.",
	})
}
