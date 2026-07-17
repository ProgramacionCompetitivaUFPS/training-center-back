package group

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Send targeted invitations by nickname (batch, best-effort)
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body inviteByNicknamesReq true "List of nicknames to invite"
// @Success      200 {object} inviteByNicknamesResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations/targeted [post]
func (h *Handler) InviteByNicknames(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var body inviteByNicknamesReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid request body"))
		return
	}

	out, err := h.inviteByNicknames.Execute(r.Context(), appGroup.InviteByNicknamesInput{
		GroupID:     r.PathValue("groupId"),
		Nicknames:   body.Nicknames,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildInviteByNicknamesResp(out))
}

func buildInviteByNicknamesResp(out *appGroup.InviteByNicknamesOutput) inviteByNicknamesResponse {
	results := make([]inviteByNicknamesResultResp, 0, len(out.Results))
	for _, r := range out.Results {
		item := inviteByNicknamesResultResp{
			Nickname: r.Nickname,
			Status:   string(r.Status),
			Reason:   r.Reason,
		}
		if r.Invitation != nil {
			item.Invitation = &invitationSummaryResp{
				ID:        r.Invitation.ID,
				Status:    r.Invitation.Status,
				ExpiresAt: r.Invitation.ExpiresAt.UTC().Format(time.RFC3339),
				CreatedAt: r.Invitation.CreatedAt.UTC().Format(time.RFC3339),
			}
		}
		if r.Invitee != nil {
			item.Invitee = &requesterResp{
				UserID:   r.Invitee.ID,
				Nickname: r.Invitee.Nickname,
				Name:     r.Invitee.Name,
				Email:    r.Invitee.Email,
			}
		}
		results = append(results, item)
	}
	return inviteByNicknamesResponse{Results: results}
}
