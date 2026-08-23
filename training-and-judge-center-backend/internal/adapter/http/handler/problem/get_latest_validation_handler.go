package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type latestValidationResponse struct {
	Found             bool                   `json:"found"`
	Terminal          bool                   `json:"terminal"`
	Passed            bool                   `json:"passed"`
	Status            string                 `json:"status,omitempty"`
	ValidationLogs    []string               `json:"validationLogs,omitempty"`
	ValidationSummary *validationSummaryResp `json:"validationSummary,omitempty"`
	FailedTestCases   []failedTestCaseResp   `json:"failedTestCases,omitempty"`
	CompilationErrors *compilationErrorsResp `json:"compilationErrors,omitempty"`
	FailedInputs      []failedInputResp      `json:"failedInputs,omitempty"`
}

// @Summary      Get the latest validation attempt for a problem
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} latestValidationResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/validation [get]
func (h *Handler) GetLatestValidation(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	out, err := h.getLatestProblemValidation.Execute(r.Context(), appProblem.GetLatestProblemValidationInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, latestValidationResponse{
		Found:             out.Found,
		Terminal:          out.Terminal,
		Passed:            out.Passed,
		Status:            out.Status,
		ValidationLogs:    out.Report.ValidationLogs,
		ValidationSummary: toValidationSummaryResp(out.Report.ValidationSummary),
		FailedTestCases:   toFailedTestCaseResps(out.Report.FailedTestCases),
		CompilationErrors: toCompilationErrorsResp(out.Report.CompilationErrors),
		FailedInputs:      toFailedInputResps(out.Report.FailedInputs),
	})
}
