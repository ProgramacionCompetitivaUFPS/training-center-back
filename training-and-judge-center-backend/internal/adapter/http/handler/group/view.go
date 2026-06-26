package group

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      Get group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      200 {object} getGroupResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId} [get]
func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getGroup.Execute(r.Context(), appGroup.GetGroupInput{
		GroupID:     r.PathValue("groupId"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	g := out.Group
	leads := make([]leadResp, 0, len(out.Leads))
	for _, l := range out.Leads {
		leads = append(leads, leadResp{UserID: l.UserID, Nickname: l.Nickname, Name: l.Name})
	}

	var joinedAt *string
	if out.Membership.JoinedAt != nil {
		s := out.Membership.JoinedAt.UTC().Format(time.RFC3339)
		joinedAt = &s
	}
	um := userMembershipResp{
		IsMember: out.Membership.Role != nil,
		Role:     out.Membership.Role,
		JoinedAt: joinedAt,
	}

	resp := getGroupResponse{
		ID:             g.ID,
		Name:           g.Name,
		Description:    g.Description,
		Visibility:     g.Visibility,
		JoinPolicy:     g.JoinPolicy,
		IsGlobal:       g.IsDefault,
		Statistics:     statisticsResp{MemberCount: out.Statistics.MemberCount, LeadCount: out.Statistics.LeadCount},
		Leads:          leads,
		UserMembership: um,
		CreatedAt:      g.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      g.UpdatedAt.UTC().Format(time.RFC3339),
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, resp)
}
