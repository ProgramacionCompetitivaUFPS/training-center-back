package team

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type registerTeamToContestRequest struct {
	SelectedMembers []string `json:"selectedMembers"`
}

type selectedMemberResponse struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type teamRegistrationResponse struct {
	ID              string                   `json:"id"`
	ContestID       string                   `json:"contestId"`
	TeamID          string                   `json:"teamId"`
	TeamName        string                   `json:"teamName"`
	SelectedMembers []selectedMemberResponse `json:"selectedMembers"`
	RegisteredAt    string                   `json:"registeredAt"`
}

// @Summary      Register team to contest
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Param        teamId path string true "Team ID"
// @Param        body body registerTeamToContestRequest true "Selected members"
// @Success      201 {object} teamRegistrationResponse
// @Failure      400,401,403,404,409 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/team-registrations/{teamId} [post]
func (h *Handler) RegisterTeamToContest(w http.ResponseWriter, r *http.Request) {
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

	var body registerTeamToContestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.registerTeamToContest.Execute(r.Context(), appTeam.RegisterTeamToContestInput{
		CurrentUser:     *caller,
		ContestID:       contestID,
		TeamID:          teamID,
		SelectedMembers: body.SelectedMembers,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, toTeamRegistrationResponse(out))
}

func toTeamRegistrationResponse(out *appTeam.RegisterTeamToContestOutput) teamRegistrationResponse {
	members := make([]selectedMemberResponse, len(out.SelectedMembers))
	for i, m := range out.SelectedMembers {
		members[i] = selectedMemberResponse{ID: m.ID, Nickname: m.Nickname}
	}
	return teamRegistrationResponse{
		ID:              out.ID,
		ContestID:       out.ContestID,
		TeamID:          out.TeamID,
		TeamName:        out.TeamName,
		SelectedMembers: members,
		RegisteredAt:    out.RegisteredAt.UTC().Format(time.RFC3339),
	}
}
