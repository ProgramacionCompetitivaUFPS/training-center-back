package submission

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	submitSolution             *appsubmission.SubmitSolutionUseCase
	submitContestSolution      *appsubmission.SubmitContestSolutionUseCase
	getSubmission              *appsubmission.GetSubmissionUseCase
	updateSubmissionVisibility *appsubmission.UpdateSubmissionVisibilityUseCase
	listMySubmissions          *appsubmission.ListMySubmissionsUseCase
	listProblemSubmissions     *appsubmission.ListProblemSubmissionsUseCase
	rejudgeSubmission          *appsubmission.RejudgeSubmissionUseCase
}

func NewHandler(
	submitSolution *appsubmission.SubmitSolutionUseCase,
	submitContestSolution *appsubmission.SubmitContestSolutionUseCase,
	getSubmission *appsubmission.GetSubmissionUseCase,
	updateSubmissionVisibility *appsubmission.UpdateSubmissionVisibilityUseCase,
	listMySubmissions *appsubmission.ListMySubmissionsUseCase,
	listProblemSubmissions *appsubmission.ListProblemSubmissionsUseCase,
	rejudgeSubmission *appsubmission.RejudgeSubmissionUseCase,
) *Handler {
	return &Handler{
		submitSolution:             submitSolution,
		submitContestSolution:      submitContestSolution,
		getSubmission:              getSubmission,
		updateSubmissionVisibility: updateSubmissionVisibility,
		listMySubmissions:          listMySubmissions,
		listProblemSubmissions:     listProblemSubmissions,
		rejudgeSubmission:          rejudgeSubmission,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*appshared.CurrentUser, bool) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid or missing authentication token"))
		return nil, false
	}
	return &cu, true
}
