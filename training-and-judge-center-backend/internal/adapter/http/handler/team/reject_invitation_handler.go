package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

// @Summary      Reject team invitation
// @Tags         teams
// @Security     BearerAuth
// @Param        invitationId path string true "Invitation ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /team-invitations/{invitationId} [delete]
func (h *Handler) RejectInvitation(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	invitationID := chi.URLParam(r, "invitationId")

	if err := h.rejectInvitation.Execute(r.Context(), appTeam.RejectInvitationInput{
		CurrentUser:  *currentUser,
		InvitationID: invitationID,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
