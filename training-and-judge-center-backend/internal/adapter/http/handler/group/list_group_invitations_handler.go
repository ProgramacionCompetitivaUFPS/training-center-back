package group

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      List invitations for a group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        status query string false "Filter by status (PENDING, ACCEPTED, REVOKED, EXPIRED); defaults to PENDING"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} listGroupInvitationsResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations [get]
func (h *Handler) ListGroupInvitations(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	q := r.URL.Query()

	page, limit, ok := parsePaginationParams(r.Context(), q, w)
	if !ok {
		return
	}

	out, err := h.listGroupInvitations.Execute(r.Context(), appGroup.ListGroupInvitationsInput{
		GroupID:     groupID,
		Status:      q.Get("status"),
		Page:        page,
		Limit:       limit,
		CurrentUser: *caller,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]invitationListItemResp, 0, len(out.Invitations))
	for _, detail := range out.Invitations {
		items = append(items, buildInvitationListItemResp(detail))
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listGroupInvitationsResponse{
		Invitations: items,
		Pagination:  buildPagination(out.Total, page, out.TotalPages, limit),
	})
}

func buildInvitationListItemResp(detail appGroup.InvitationDetail) invitationListItemResp {
	resp := invitationListItemResp{
		ID:              detail.Invitation.ID,
		GroupID:         detail.Invitation.GroupID,
		Status:          detail.Invitation.Status,
		EffectiveStatus: detail.EffectiveStatus,
		ExpiresAt:       detail.Invitation.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:       detail.Invitation.CreatedAt.UTC().Format(time.RFC3339),
	}
	if detail.InvitedBy != nil {
		resp.InvitedBy = detail.InvitedBy.Nickname
	}
	if detail.Invitee != nil {
		resp.Invitee = &requesterResp{
			UserID:   detail.Invitee.ID,
			Nickname: detail.Invitee.Nickname,
			Name:     detail.Invitee.Name,
			Email:    detail.Invitee.Email,
		}
	}
	return resp
}
