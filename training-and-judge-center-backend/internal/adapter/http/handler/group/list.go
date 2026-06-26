package group

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      List groups
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Param        search query string false "Search term"
// @Param        visibility query string false "Filter by visibility"
// @Param        joinPolicy query string false "Filter by join policy"
// @Success      200 {object} listGroupsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /groups [get]
func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	page, limit, ok := parsePaginationParams(r.Context(), q, w)
	if !ok {
		return
	}

	in := appGroup.ListGroupsInput{
		CurrentUser: *currentUser,
		Search:      q.Get("search"),
		Visibility:  stringPtrOrNil(q.Get("visibility")),
		JoinPolicy:  stringPtrOrNil(q.Get("joinPolicy")),
		SortBy:      q.Get("sortBy"),
		Order:       q.Get("order"),
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listGroups.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]groupListItemResp, 0, len(out.Groups))
	for _, lg := range out.Groups {
		g := lg.Group
		items = append(items, groupListItemResp{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			Visibility:  g.Visibility,
			JoinPolicy:  g.JoinPolicy,
			IsGlobal:    g.IsDefault,
			MemberCount: lg.MemberCount,
			UserRole:    stringPtrOrNil(lg.UserRole),
			CreatedAt:   g.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listGroupsResponse{
		Groups:     items,
		Pagination: buildPagination(out.TotalCount, out.Page, out.TotalPages, out.Limit),
	})
}
