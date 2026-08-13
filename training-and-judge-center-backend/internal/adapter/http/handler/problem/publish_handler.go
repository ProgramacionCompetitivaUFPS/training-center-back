package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type publishResponse struct {
	Slug              string                        `json:"slug"`
	Status            string                        `json:"status"`
	Message           string                        `json:"message"`
	ValidationLogs    []string                      `json:"validationLogs"`
	ValidationSummary *appProblem.ValidationSummary `json:"validationSummary,omitempty"`
}

type publishFailureResponse struct {
	Error             string                        `json:"error"`
	Message           string                        `json:"message"`
	ValidationLogs    []string                      `json:"validationLogs"`
	MissingFields     []string                      `json:"missingFields,omitempty"`
	FailedTestCases   []appProblem.FailedTestCase   `json:"failedTestCases,omitempty"`
	CompilationErrors *appProblem.CompilationErrors `json:"compilationErrors,omitempty"`
	FailedInputs      []appProblem.FailedInput      `json:"failedInputs,omitempty"`
}

// @Summary      Publish problem
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} publishResponse
// @Failure      400 {object} publishFailureResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      503 {object} apperror.AppError
// @Router       /problems/p/{slug}/publish [post]
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	out, err := h.publishProblem.Execute(r.Context(), appProblem.PublishProblemInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if len(out.MissingFields) > 0 {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, publishFailureResponse{
			Error:          "VALIDATION_FAILED",
			Message:        "Problem validation failed",
			ValidationLogs: out.ValidationLogs,
			MissingFields:  out.MissingFields,
		})
		return
	}

	status, err := h.awaitProblemValidation.Execute(r.Context(), appProblem.AwaitProblemValidationInput{ValidationID: out.ValidationID})
	if err != nil {
		if r.Context().Err() != nil {
			return // client disconnected — nothing left to write to
		}
		handler.WriteError(r.Context(), w, err)
		return
	}
	writePublishOutcome(w, r, slug, status)
}

func writePublishOutcome(w http.ResponseWriter, r *http.Request, slug string, out *appProblem.GetProblemValidationStatusOutput) {
	if out.Passed {
		handler.WriteJSON(r.Context(), w, http.StatusOK, publishResponse{
			Slug:              slug,
			Status:            out.Status,
			Message:           "Problem published successfully",
			ValidationLogs:    out.Report.ValidationLogs,
			ValidationSummary: out.Report.ValidationSummary,
		})
		return
	}
	handler.WriteJSON(r.Context(), w, http.StatusBadRequest, publishFailureResponse{
		Error:             "VALIDATION_FAILED",
		Message:           validationFailureMessage(out.Report),
		ValidationLogs:    out.Report.ValidationLogs,
		FailedTestCases:   out.Report.FailedTestCases,
		CompilationErrors: out.Report.CompilationErrors,
		FailedInputs:      out.Report.FailedInputs,
	})
}

func validationFailureMessage(r appProblem.ValidationReport) string {
	switch {
	case r.CompilationErrors != nil:
		return "Compilation failed"
	case len(r.FailedInputs) > 0:
		return "Validator rejected test inputs"
	case len(r.FailedTestCases) > 0:
		return "Solution failed test cases"
	default:
		return "Problem validation failed"
	}
}
