package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

// @Summary      Leave team
// @Tags         teams
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /teams/{teamId}/members/me [delete]
func (h *Handler) LeaveTeam(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	teamID := chi.URLParam(r, "teamId")

	if err := h.leaveTeam.Execute(r.Context(), appTeam.LeaveTeamInput{
		CurrentUser: *currentUser,
		TeamID:      teamID,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
