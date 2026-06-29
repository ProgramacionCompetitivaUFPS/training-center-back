package submission

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// @Summary      Admin rejudge specific submission
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        submissionId path string true "Submission ID"
// @Success      200 {object} rejudgeSubmissionResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /admin/submissions/{submissionId}/rejudge [post]
func (h *Handler) AdminRejudgeSubmission(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	submissionID := r.PathValue("submissionId")

	out, err := h.rejudgeSubmission.Execute(r.Context(), appsubmission.RejudgeSubmissionInput{
		SubmissionID: submissionID,
		CurrentUser:  *cu,
		Now:          time.Now(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, rejudgeSubmissionResponse{
		SubmissionID:    out.SubmissionID,
		ProblemSlug:     out.ProblemSlug,
		PreviousVerdict: out.PreviousVerdict,
		CurrentStatus:   "PENDING",
		Message:         "Admin submission rejudge initiated successfully",
	})
}
