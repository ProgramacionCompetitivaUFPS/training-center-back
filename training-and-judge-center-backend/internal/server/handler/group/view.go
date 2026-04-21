package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	groupID := r.PathValue("groupId")
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	out, err := h.getUC.Execute(r.Context(), appGroup.GetGroupInput{
		GroupID:     groupID,
		CurrentUser: currentUser,
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

	um := userMembershipResp{IsMember: out.Membership.IsMember, JoinedAt: out.Membership.JoinedAt}
	if out.Membership.Role != nil {
		s := string(*out.Membership.Role)
		um.Role = &s
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
		CreatedAt:      g.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      g.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
