package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type leaveImpactRegistrationResp struct {
	ContestID   string `json:"contestId"`
	ContestName string `json:"contestName"`
	GroupID     string `json:"groupId"`
}

type leaveImpactResponse struct {
	RegistrationsToCancel []leaveImpactRegistrationResp `json:"registrationsToCancel"`
}

// @Summary      What leaving the team would cost it
// @Description  Lists the team's registrations in contests that have not started that would be cancelled if the caller left, because the team would be left with fewer players than the contest's minimum team size. Leaving does that cancellation itself; this only tells the member beforehand.
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Success      200 {object} leaveImpactResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /teams/{teamId}/leave-impact [get]
func (h *Handler) GetLeaveImpact(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getLeaveImpact.Execute(r.Context(), appTeam.GetLeaveImpactInput{
		CurrentUser: *currentUser,
		TeamID:      chi.URLParam(r, "teamId"),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	resp := leaveImpactResponse{RegistrationsToCancel: make([]leaveImpactRegistrationResp, 0, len(out.RegistrationsToCancel))}
	for _, reg := range out.RegistrationsToCancel {
		resp.RegistrationsToCancel = append(resp.RegistrationsToCancel, leaveImpactRegistrationResp{
			ContestID: reg.ContestID, ContestName: reg.ContestName, GroupID: reg.GroupID,
		})
	}
	handler.WriteJSON(r.Context(), w, http.StatusOK, resp)
}
