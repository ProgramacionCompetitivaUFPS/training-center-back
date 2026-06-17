package team

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type teamMemberItem struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	JoinedAt string `json:"joinedAt"`
}

type teamCreator struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type getTeamResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	CreatedBy teamCreator      `json:"createdBy"`
	CreatedAt string           `json:"createdAt"`
	Members   []teamMemberItem `json:"members"`
}

// @Summary      Get team details
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Success      200 {object} getTeamResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /teams/{teamId} [get]
func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	teamID := chi.URLParam(r, "teamId")

	out, err := h.getTeam.Execute(r.Context(), appTeam.GetTeamInput{
		TeamID: teamID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	members := make([]teamMemberItem, len(out.Members))
	for i, m := range out.Members {
		members[i] = teamMemberItem{
			ID:       m.UserID,
			Nickname: m.Nickname,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, getTeamResponse{
		ID:   out.ID,
		Name: out.Name,
		CreatedBy: teamCreator{
			ID:       out.CreatedBy.UserID,
			Nickname: out.CreatedBy.Nickname,
		},
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339),
		Members:   members,
	})
}
