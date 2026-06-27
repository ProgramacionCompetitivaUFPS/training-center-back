package contest

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Get contest
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId   path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Success      200 {object} getContestResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	out, err := h.getContest.Execute(r.Context(), appContest.GetContestInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toGetContestResponse(out))
}

func toGetContestResponse(out *appContest.GetContestOutput) getContestResponse {
	probs := make([]problemDetail, len(out.Problems))
	for i, p := range out.Problems {
		probs[i] = problemDetail{
			Position:    p.Position,
			Slug:        p.Slug,
			Title:       p.Title,
			TimeLimit:   p.TimeLimit,
			MemoryLimit: p.MemoryLimit,
		}
	}

	resp := getContestResponse{
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
		ParticipantCount:  out.ParticipantCount,
		IsRegistered:      out.IsRegistered,
		Group:             groupDisplay{ID: out.Group.ID, Name: out.Group.Name},
		Owner:             ownerDisplay{ID: out.Owner.ID, Nickname: out.Owner.Nickname, Name: out.Owner.Name},
		Problems:          probs,
		Status:            out.Status,
		CreatedAt:         out.CreatedAt.UTC().Format(time.RFC3339),
	}
	if out.UpdatedAt != nil {
		s := out.UpdatedAt.UTC().Format(time.RFC3339)
		resp.UpdatedAt = &s
	}
	return resp
}
