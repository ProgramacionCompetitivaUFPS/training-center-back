package team

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type myTeamItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"memberCount"`
	JoinedAt    string `json:"joinedAt"`
	CreatedAt   string `json:"createdAt"`
}

type listMyTeamsResponse struct {
	Teams      []myTeamItem   `json:"teams"`
	Pagination paginationMeta `json:"pagination"`
}

type paginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// @Summary      List my teams
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} listMyTeamsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/teams [get]
func (h *Handler) ListMyTeams(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)

	out, err := h.listMyTeams.Execute(r.Context(), appTeam.ListMyTeamsInput{
		CurrentUser: *currentUser,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]myTeamItem, len(out.Teams))
	for i, t := range out.Teams {
		items[i] = myTeamItem{
			ID:          t.ID,
			Name:        t.Name,
			MemberCount: t.MemberCount,
			JoinedAt:    t.JoinedAt.UTC().Format(time.RFC3339),
			CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listMyTeamsResponse{
		Teams: items,
		Pagination: paginationMeta{
			Page:       out.Page,
			Limit:      out.Limit,
			Total:      out.TotalCount,
			TotalPages: out.TotalPages,
		},
	})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
