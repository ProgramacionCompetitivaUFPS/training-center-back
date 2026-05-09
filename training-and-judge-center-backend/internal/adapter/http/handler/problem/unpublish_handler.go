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
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	out, err := h.unpublishUC.Execute(r.Context(), appProblem.UnpublishProblemInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, unpublishResponse{
		Slug:    out.Problem.Slug,
		Status:  out.Problem.Status,
		Message: "Problem unpublished successfully. You can now make changes.",
	})
}
