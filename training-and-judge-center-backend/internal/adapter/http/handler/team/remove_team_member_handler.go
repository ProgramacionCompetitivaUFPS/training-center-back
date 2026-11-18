package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

// @Summary      Remove a team member
// @Description  Any member of the team can remove another member. It is refused while that member is selected in a running contest, and it takes them out of the registrations of contests that have not started (cancelling those left below the contest's minimum team size). To remove yourself, leave the team instead.
// @Tags         teams
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Param        userId path string true "ID of the member to remove"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /teams/{teamId}/members/{userId} [delete]
func (h *Handler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	if err := h.removeTeamMember.Execute(r.Context(), appTeam.RemoveTeamMemberInput{
		CurrentUser:  *currentUser,
		TeamID:       chi.URLParam(r, "teamId"),
		TargetUserID: chi.URLParam(r, "userId"),
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
