package team

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type inviteToTeamRequest struct {
	Nickname string `json:"nickname"`
}

type userRef struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type inviteToTeamResponse struct {
	ID          string  `json:"id"`
	TeamID      string  `json:"teamId"`
	InviteeUser userRef `json:"inviteeUser"`
	InvitedBy   userRef `json:"invitedBy"`
	CreatedAt   string  `json:"createdAt"`
}

// @Summary      Invite user to team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Param        body body inviteToTeamRequest true "Invite request"
// @Success      201 {object} inviteToTeamResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /teams/{teamId}/invitations [post]
func (h *Handler) InviteToTeam(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	teamID := chi.URLParam(r, "teamId")

	var req inviteToTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}
	if req.Nickname == "" {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: "nickname is required"},
		}))
		return
	}

	out, err := h.inviteToTeam.Execute(r.Context(), appTeam.InviteToTeamInput{
		CurrentUser: *currentUser,
		TeamID:      teamID,
		Nickname:    req.Nickname,
	}, time.Now())
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, inviteToTeamResponse{
		ID:          out.ID,
		TeamID:      out.TeamID,
		InviteeUser: userRef{ID: out.InviteeUser.ID, Nickname: out.InviteeUser.Nickname},
		InvitedBy:   userRef{ID: out.InvitedBy.ID, Nickname: out.InvitedBy.Nickname},
		CreatedAt:   out.CreatedAt.UTC().Format(time.RFC3339),
	})
}
