package team

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type acceptInvitationResponse struct {
	JoinedAt string `json:"joinedAt"`
}

// @Summary      Accept team invitation
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        invitationId path string true "Invitation ID"
// @Success      200 {object} acceptInvitationResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /team-invitations/{invitationId}/accept [post]
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	invitationID := chi.URLParam(r, "invitationId")

	out, err := h.acceptInvitation.Execute(r.Context(), appTeam.AcceptInvitationInput{
		CurrentUser:  *currentUser,
		InvitationID: invitationID,
	}, time.Now())
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, acceptInvitationResponse{
		JoinedAt: out.JoinedAt.UTC().Format(time.RFC3339),
	})
}
