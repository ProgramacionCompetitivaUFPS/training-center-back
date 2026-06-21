package submission

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type getSubmissionResponse struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	Visibility    string          `json:"visibility"`
	SubmittedAt   string          `json:"submittedAt"`
	JudgedAt      *string         `json:"judgedAt"`
	Problem       problemSummary  `json:"problem"`
	Contest       *contestSummary `json:"contest"`
	SubmittedBy   userSummary     `json:"submittedBy"`
	Language      string          `json:"language"`
	Compiler      string          `json:"compiler"`
	ExecutionTime *int            `json:"executionTime"`
	MemoryUsed    *int            `json:"memoryUsed"`
	SourceCode    string          `json:"sourceCode"`
}

// GetSubmission handles GET /submissions/{submissionId}
//
// @Summary      Get submission
// @Tags         submissions
// @Security     BearerAuth
// @Produce      json
// @Param        submissionId path string true "Submission ID"
// @Success      200 {object} getSubmissionResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /submissions/{submissionId} [get]
func (h *Handler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	cu, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	submissionID := r.PathValue("submissionId")
	if submissionID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "submissionId is required"))
		return
	}

	out, err := h.getSubmission.Execute(r.Context(), appsubmission.GetSubmissionInput{
		CurrentUser:  *cu,
		SubmissionID: submissionID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	resp := getSubmissionResponse{
		ID:          out.ID,
		Status:      out.Status,
		Visibility:  out.Visibility,
		SubmittedAt: out.SubmittedAt.UTC().Format(time.RFC3339),
		Problem:     problemSummary{Slug: out.Problem.Slug, Title: out.Problem.Title},
		SubmittedBy: userSummary{ID: out.SubmittedBy.ID, Nickname: out.SubmittedBy.Nickname},
		Language:    out.Language,
		Compiler:    out.Compiler,
		ExecutionTime: out.ExecutionTime,
		MemoryUsed:    out.MemoryUsed,
		SourceCode:  out.SourceCode,
	}

	if out.JudgedAt != nil {
		s := out.JudgedAt.UTC().Format(time.RFC3339)
		resp.JudgedAt = &s
	}

	if out.Contest != nil {
		resp.Contest = &contestSummary{ID: out.Contest.ID, Name: out.Contest.Name}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, resp)
}
