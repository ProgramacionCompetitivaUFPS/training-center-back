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

// @Summary      Generate a group invitation
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body generateInviteReq false "Optional invitee identifier (at most one of userNickname, userEmail, userId); omit for a general link-style invitation"
// @Success      201 {object} generateInviteResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations [post]
func (h *Handler) GenerateInvite(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var body generateInviteReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid request body"))
		return
	}

	out, err := h.generateInvite.Execute(r.Context(), appGroup.GenerateInviteInput{
		GroupID:      r.PathValue("groupId"),
		UserNickname: body.UserNickname,
		UserEmail:    body.UserEmail,
		UserID:       body.UserID,
		CurrentUser:  *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, buildGenerateInviteResp(out))
}

func buildGenerateInviteResp(out *appGroup.GenerateInviteOutput) generateInviteResponse {
	resp := generateInviteResponse{
		ID:        out.Invitation.ID,
		GroupID:   out.Invitation.GroupID,
		Status:    out.Invitation.Status,
		ExpiresAt: out.Invitation.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt: out.Invitation.CreatedAt.UTC().Format(time.RFC3339),
	}
	if out.Invitee != nil {
		resp.Invitee = &requesterResp{
			UserID:   out.Invitee.ID,
			Nickname: out.Invitee.Nickname,
			Name:     out.Invitee.Name,
			Email:    out.Invitee.Email,
		}
	}
	return resp
}
