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

type adminRejudgeResponse struct {
	ProblemSlug        string  `json:"problemSlug"`
	SubmissionsQueued  int     `json:"submissionsQueued"`
	ContestStatus      *string `json:"contestStatus,omitempty"`
	StandingWillUpdate *bool   `json:"standingWillUpdate,omitempty"`
	Message            string  `json:"message"`
}

// @Summary      Admin rejudge problem submissions
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        slug      path  string true  "Problem slug"
// @Param        contestId query string false "Filter by contest ID"
// @Success      200 {object} adminRejudgeResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /admin/problems/{slug}/rejudge [post]
func (h *Handler) AdminRejudge(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var contestID *string
	if cid := r.URL.Query().Get("contestId"); cid != "" {
		contestID = &cid
	}

	out, err := h.adminRejudgeSubmissions.Execute(r.Context(), appProblem.AdminRejudgeSubmissionsInput{
		Slug:        slug,
		ContestID:   contestID,
		CurrentUser: shared.CurrentUser{ID: cu.ID, Role: cu.Role},
		Now:         time.Now(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, adminRejudgeResponse{
		ProblemSlug:        out.ProblemSlug,
		SubmissionsQueued:  out.SubmissionsQueued,
		ContestStatus:      out.ContestStatus,
		StandingWillUpdate: out.StandingWillUpdate,
		Message:            "Admin rejudge initiated successfully",
	})
}
