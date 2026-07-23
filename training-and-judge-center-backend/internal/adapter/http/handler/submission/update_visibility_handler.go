package submission

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type updateVisibilityRequest struct {
	Visibility string `json:"visibility"`
}

// UpdateVisibility handles PATCH /submissions/{submissionId}/visibility
//
// @Summary      Update submission visibility
// @Tags         submissions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        submissionId path   string                   true "Submission ID"
// @Param        body         body   updateVisibilityRequest  true "New visibility"
// @Success      200
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /submissions/{submissionId}/visibility [patch]
func (h *Handler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	submissionID := r.PathValue("submissionId")
	if submissionID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "submissionId is required"))
		return
	}

	var req updateVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid request body"))
		return
	}

	if err := h.updateSubmissionVisibility.Execute(r.Context(), appsubmission.UpdateSubmissionVisibilityInput{
		CurrentUser:   *cu,
		SubmissionID:  submissionID,
		NewVisibility: req.Visibility,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, nil)
}
