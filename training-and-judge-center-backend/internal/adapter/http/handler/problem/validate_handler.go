package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type validateResponse struct {
	Slug         string `json:"slug"`
	ValidationID string `json:"validationId"`
	Message      string `json:"message"`
}

// @Summary      Validate a draft problem without publishing it
// @Description  Runs the same validation as publishing (compiles the checker and validator, runs every reference solution against every test case) but leaves the problem a draft. It answers 202 right away; follow the result with GET /problems/p/{slug}/validation. Publishing afterwards reuses a passing validation as long as the problem has not changed since.
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      202 {object} validateResponse
// @Failure      400 {object} publishFailureResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /problems/p/{slug}/validate [post]
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	out, err := h.requestProblemValidation.Execute(r.Context(), appProblem.RequestProblemValidationInput{
		Slug:        slug,
		CurrentUser: shared.CurrentUser{ID: cu.ID, Role: cu.Role},
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

	handler.WriteJSON(r.Context(), w, http.StatusAccepted, validateResponse{
		Slug:         out.Slug,
		ValidationID: out.ValidationID,
		Message:      "Validation started",
	})
}
