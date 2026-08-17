package user

import (
	"net/http"
	"strconv"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type userSearchResultResp struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type searchUsersResponse struct {
	Users []userSearchResultResp `json:"users"`
}

// @Summary      Search users (autocomplete, Coach/Admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        q     query string true  "Search query (min 2 characters)"
// @Param        limit query int    false "Max results (default 10, max 20)"
// @Success      200 {object} searchUsersResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /users/search [get]
func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	out, err := h.searchUsers.Execute(r.Context(), appuser.SearchUsersInput{
		Query: r.URL.Query().Get("q"),
		Limit: limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	users := make([]userSearchResultResp, len(out.Users))
	for i, u := range out.Users {
		users[i] = userSearchResultResp{ID: u.ID, Nickname: u.Nickname, Name: u.Name}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, searchUsersResponse{Users: users})
}
