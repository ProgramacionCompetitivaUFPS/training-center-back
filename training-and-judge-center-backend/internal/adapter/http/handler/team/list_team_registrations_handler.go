package team

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type teamRegistrationItem struct {
	ID              string                   `json:"id"`
	TeamID          string                   `json:"teamId"`
	TeamName        string                   `json:"teamName"`
	SelectedMembers []selectedMemberResponse `json:"selectedMembers"`
	RegisteredAt    string                   `json:"registeredAt"`
}

type listTeamRegistrationsResponse struct {
	Items      []teamRegistrationItem `json:"items"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"totalPages"`
}

// @Summary      List team registrations for a contest
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Param        page query int false "Page" default(1)
// @Param        limit query int false "Limit" default(20)
// @Success      200 {object} listTeamRegistrationsResponse
// @Failure      401,403,404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/team-registrations [get]
func (h *Handler) ListTeamRegistrations(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	page := 1
	limit := 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	out, err := h.listTeamRegistrations.Execute(r.Context(), appTeam.ListTeamRegistrationsInput{
		CurrentUser: *caller,
		ContestID:   contestID,
		GroupID:     groupID,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]teamRegistrationItem, len(out.Items))
	for i, item := range out.Items {
		members := make([]selectedMemberResponse, len(item.SelectedMembers))
		for j, m := range item.SelectedMembers {
			members[j] = selectedMemberResponse{ID: m.ID, Nickname: m.Nickname}
		}
		items[i] = teamRegistrationItem{
			ID:              item.ID,
			TeamID:          item.TeamID,
			TeamName:        item.TeamName,
			SelectedMembers: members,
			RegisteredAt:    item.RegisteredAt.UTC().Format(time.RFC3339),
		}
	}

	totalPages := out.Total / limit
	if out.Total%limit != 0 {
		totalPages++
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listTeamRegistrationsResponse{
		Items:      items,
		Total:      out.Total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}
