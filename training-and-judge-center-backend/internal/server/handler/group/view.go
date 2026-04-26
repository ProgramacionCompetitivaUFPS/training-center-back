package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getUC.Execute(r.Context(), appGroup.GetGroupInput{
		GroupID:     r.PathValue("groupId"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	g := out.Group
	leads := make([]leadResp, 0, len(out.Leads))
	for _, l := range out.Leads {
		leads = append(leads, leadResp{UserID: l.UserID, Nickname: l.Nickname, Name: l.Name})
	}

	var joinedAt *string
	if out.Membership.JoinedAt != nil {
		s := out.Membership.JoinedAt.Format(timeutil.RFC3339UTC)
		joinedAt = &s
	}
	um := userMembershipResp{
		IsMember: out.Membership.IsMember,
		Role:     memberRoleToStringPtr(out.Membership.Role),
		JoinedAt: joinedAt,
	}

	resp := getGroupResponse{
		ID:             g.ID(),
		Name:           g.Name().String(),
		Description:    g.Description(),
		Visibility:     g.Visibility().String(),
		JoinPolicy:     g.JoinPolicy().String(),
		IsGlobal:       g.IsDefault(),
		Statistics:     statisticsResp{MemberCount: out.Statistics.MemberCount, LeadCount: out.Statistics.LeadCount},
		Leads:          leads,
		UserMembership: um,
		CreatedAt:      g.CreatedAt().Format(timeutil.RFC3339UTC),
		UpdatedAt:      g.UpdatedAt().Format(timeutil.RFC3339UTC),
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
