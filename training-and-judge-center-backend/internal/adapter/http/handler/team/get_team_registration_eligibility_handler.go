package team

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type memberEligibilityResponse struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Eligible bool   `json:"eligible"`
	// Reason is the error code registration would fail with for this member:
	// MEMBER_NOT_IN_GROUP, MEMBER_ALREADY_REGISTERED or MEMBER_IN_ANOTHER_TEAM.
	// Omitted when the member is eligible.
	Reason string `json:"reason,omitempty"`
}

type teamRegistrationEligibilityResponse struct {
	TeamAlreadyRegistered bool                        `json:"teamAlreadyRegistered"`
	Members               []memberEligibilityResponse `json:"members"`
}

// @Summary      Get which team members can be selected for a contest registration
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Param        teamId path string true "Team ID"
// @Success      200 {object} teamRegistrationEligibilityResponse
// @Failure      401,403,404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/team-registrations/{teamId}/eligibility [get]
func (h *Handler) GetTeamRegistrationEligibility(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	teamID := r.PathValue("teamId")
	if groupID == "" || contestID == "" || teamID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	out, err := h.getTeamRegistrationEligibility.Execute(r.Context(), appTeam.GetTeamRegistrationEligibilityInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		TeamID:      teamID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	members := make([]memberEligibilityResponse, len(out.Members))
	for i, m := range out.Members {
		members[i] = memberEligibilityResponse{
			ID:       m.ID,
			Nickname: m.Nickname,
			Eligible: m.Eligible,
			Reason:   m.Reason,
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, teamRegistrationEligibilityResponse{
		TeamAlreadyRegistered: out.TeamAlreadyRegistered,
		Members:               members,
	})
}
