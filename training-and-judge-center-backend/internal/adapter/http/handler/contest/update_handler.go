package contest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Update contest
// @Tags         contests
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Param        body body updateContestRequest true "Fields to update"
// @Success      200 {object} contestResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	var body updateContestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	in := appContest.UpdateContestInput{
		CurrentUser:       *caller,
		GroupID:           groupID,
		ContestID:         contestID,
		Name:              body.Name,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		Penalty:           body.Penalty,
		FreezeMinutes:     body.FreezeMinutes,
		EnablePostContest: body.EnablePostContest,
		Locked:            body.Locked,
	}

	// description: pass **string so the use case can distinguish no-change vs clear.
	if body.Description != nil {
		in.Description = &body.Description
	}

	if body.Problems != nil {
		problems := make([]appContest.ProblemOrderInput, len(*body.Problems))
		for i, p := range *body.Problems {
			problems[i] = appContest.ProblemOrderInput{Slug: p.Slug, Order: p.Order}
		}
		in.Problems = &problems
	}

	out, err := h.updateContest.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toContestResponse(out))
}

func toContestResponse(out *appContest.ContestOutput) contestResponse {
	probs := make([]problemDisplay, len(out.Problems))
	for i, p := range out.Problems {
		probs[i] = problemDisplay{Slug: p.Slug, Title: p.Title, Order: p.Order}
	}

	resp := contestResponse{
		ID:                out.ID,
		Name:              out.Name,
		Description:       out.Description,
		StartTime:         out.StartTime.UTC().Format(time.RFC3339),
		EndTime:           out.EndTime.UTC().Format(time.RFC3339),
		Duration:          out.Duration,
		Penalty:           out.Penalty,
		FreezeMinutes:     out.FreezeMinutes,
		EnablePostContest: out.EnablePostContest,
		Locked:            out.Locked,
		Group:             groupDisplay{ID: out.Group.ID, Name: out.Group.Name},
		Owner:             ownerDisplay{Nickname: out.Owner.Nickname, Name: out.Owner.Name},
		Problems:          probs,
		ProblemCount:      out.ProblemCount,
		Status:            out.Status,
		CreatedAt:         out.CreatedAt.UTC().Format(time.RFC3339),
	}
	if out.UpdatedAt != nil {
		s := out.UpdatedAt.UTC().Format(time.RFC3339)
		resp.UpdatedAt = &s
	}
	return resp
}
