package team

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Unregister team from contest
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Param        teamId path string true "Team ID"
// @Success      204
// @Failure      401,403,404,409 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/team-registrations/{teamId} [delete]
func (h *Handler) UnregisterTeamFromContest(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	contestID := r.PathValue("contestId")
	teamID := r.PathValue("teamId")
	if contestID == "" || teamID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	if err := h.unregisterTeamFromContest.Execute(r.Context(), appTeam.UnregisterTeamFromContestInput{
		CurrentUser: *caller,
		ContestID:   contestID,
		TeamID:      teamID,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
