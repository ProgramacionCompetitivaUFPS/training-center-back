package problem

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type rejudgeResponse struct {
	ProblemSlug       string `json:"problemSlug"`
	SubmissionsQueued int    `json:"submissionsQueued"`
	Message           string `json:"message"`
}

// @Summary      Rejudge submissions
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} rejudgeResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/rejudge [post]
func (h *Handler) Rejudge(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	out, err := h.rejudgeSubmissions.Execute(r.Context(), appProblem.RejudgeSubmissionsInput{
		Slug:        slug,
		CurrentUser: currentUser,
		Now:         time.Now(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, rejudgeResponse{
		ProblemSlug:       out.ProblemSlug,
		SubmissionsQueued: out.SubmissionsQueued,
		Message:           "Rejudge initiated successfully",
	})
}
