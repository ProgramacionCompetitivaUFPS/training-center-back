package contest

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Create contest
// @Tags         contests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body createContestRequest true "Contest data"
// @Success      201 {object} contestResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing groupId"))
		return
	}

	var body createContestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.createContest.Execute(r.Context(), appContest.CreateContestInput{
		CurrentUser:       *caller,
		GroupID:           groupID,
		Name:              body.Name,
		Description:       body.Description,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		Penalty:           body.Penalty,
		FreezeMinutes:     body.FreezeMinutes,
		EnablePostContest: body.EnablePostContest,
		Problems:          body.Problems,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, toContestResponse(out))
}
