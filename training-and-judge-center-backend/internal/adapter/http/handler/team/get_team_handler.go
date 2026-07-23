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

type teamUserRef struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type pendingInvitationItem struct {
	ID        string      `json:"id"`
	Invitee   teamUserRef `json:"invitee"`
	InvitedBy teamUserRef `json:"invitedBy"`
	InvitedAt string      `json:"invitedAt"`
}

type getTeamResponse struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	CreatedBy          teamCreator             `json:"createdBy"`
	CreatedAt          string                  `json:"createdAt"`
	Members            []teamMemberItem        `json:"members"`
	PendingInvitations []pendingInvitationItem `json:"pendingInvitations"`
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
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	teamID := chi.URLParam(r, "teamId")

	out, err := h.getTeam.Execute(r.Context(), appTeam.GetTeamInput{
		CurrentUser: *cu,
		TeamID:      teamID,
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

	invitations := make([]pendingInvitationItem, len(out.PendingInvitations))
	for i, inv := range out.PendingInvitations {
		invitations[i] = pendingInvitationItem{
			ID:        inv.ID,
			Invitee:   teamUserRef{ID: inv.Invitee.ID, Nickname: inv.Invitee.Nickname},
			InvitedBy: teamUserRef{ID: inv.InvitedBy.ID, Nickname: inv.InvitedBy.Nickname},
			InvitedAt: inv.InvitedAt.UTC().Format(time.RFC3339),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, getTeamResponse{
		ID:   out.ID,
		Name: out.Name,
		CreatedBy: teamCreator{
			ID:       out.CreatedBy.UserID,
			Nickname: out.CreatedBy.Nickname,
		},
		CreatedAt:          out.CreatedAt.UTC().Format(time.RFC3339),
		Members:            members,
		PendingInvitations: invitations,
	})
}
