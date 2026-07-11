package group

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Accept a group invitation
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body acceptInviteRequest true "Invitation ID"
// @Success      201 {object} joinGroupResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations/accept [post]
func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var body acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid request body"))
		return
	}

	out, err := h.acceptInvite.Execute(r.Context(), appGroup.AcceptInviteInput{
		GroupID:      r.PathValue("groupId"),
		InvitationID: body.InvitationID,
		CurrentUser:  *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	m := out.Member
	handler.WriteJSON(r.Context(), w, http.StatusCreated, joinGroupResponse{
		Role:     m.Role,
		JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
	})
}
