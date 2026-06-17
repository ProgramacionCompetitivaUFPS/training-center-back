package team

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type teamRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type invitationItem struct {
	ID        string  `json:"id"`
	Team      teamRef `json:"team"`
	InvitedBy userRef `json:"invitedBy"`
	CreatedAt string  `json:"createdAt"`
}

type listMyInvitationsResponse struct {
	Invitations []invitationItem `json:"invitations"`
}

// @Summary      List my team invitations
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} listMyInvitationsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/team-invitations [get]
func (h *Handler) ListMyInvitations(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.listMyInvitations.Execute(r.Context(), appTeam.ListMyInvitationsInput{
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]invitationItem, len(out.Invitations))
	for i, inv := range out.Invitations {
		items[i] = invitationItem{
			ID:        inv.ID,
			Team:      teamRef{ID: inv.Team.ID, Name: inv.Team.Name},
			InvitedBy: userRef{ID: inv.InvitedBy.ID, Nickname: inv.InvitedBy.Nickname},
			CreatedAt: inv.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listMyInvitationsResponse{Invitations: items})
}
