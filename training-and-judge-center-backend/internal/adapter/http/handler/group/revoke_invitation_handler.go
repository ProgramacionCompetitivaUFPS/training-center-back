package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      Revoke a group invitation
// @Tags         groups
// @Security     BearerAuth
// @Param        groupId      path string true "Group ID"
// @Param        invitationId path string true "Invitation ID"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations/{invitationId} [delete]
func (h *Handler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	err := h.revokeInvitation.Execute(r.Context(), appGroup.RevokeInvitationInput{
		GroupID:      r.PathValue("groupId"),
		InvitationID: r.PathValue("invitationId"),
		CurrentUser:  *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
