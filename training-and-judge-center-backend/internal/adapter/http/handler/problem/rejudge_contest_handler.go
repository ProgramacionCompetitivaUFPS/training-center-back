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

type rejudgeContestResponse struct {
	ContestID          string `json:"contestId"`
	ProblemSlug        string `json:"problemSlug"`
	SubmissionsQueued  int    `json:"submissionsQueued"`
	ContestStatus      string `json:"contestStatus"`
	StandingWillUpdate bool   `json:"standingWillUpdate"`
	Message            string `json:"message"`
}

// @Summary      Rejudge contest submissions
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        contestId path string true "Contest ID"
// @Param        slug      path string true "Problem slug"
// @Success      200 {object} rejudgeContestResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /contests/{contestId}/problems/{slug}/rejudge [post]
func (h *Handler) RejudgeContest(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	contestID := r.PathValue("contestId")
	slug := r.PathValue("slug")

	out, err := h.rejudgeContestSubmissions.Execute(r.Context(), appProblem.RejudgeContestSubmissionsInput{
		ContestID:   contestID,
		Slug:        slug,
		CurrentUser: shared.CurrentUser{ID: cu.ID, Role: cu.Role},
		Now:         time.Now(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, rejudgeContestResponse{
		ContestID:          out.ContestID,
		ProblemSlug:        out.ProblemSlug,
		SubmissionsQueued:  out.SubmissionsQueued,
		ContestStatus:      out.ContestStatus,
		StandingWillUpdate: out.StandingWillUpdate,
		Message:            "Rejudge initiated successfully",
	})
}
