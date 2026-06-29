package submission

import (
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
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
