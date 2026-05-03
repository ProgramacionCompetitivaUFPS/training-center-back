package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/timeutil"
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
	page, limit, ok := parsePaginationParams(q, w)
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
		handler.WriteError(w, err)
		return
	}

	items := make([]groupListItemResp, 0, len(out.Groups))
	for _, lg := range out.Groups {
		g := lg.Group
		items = append(items, groupListItemResp{
			ID:          g.ID(),
			Name:        g.Name().String(),
			Description: g.Description(),
			Visibility:  g.Visibility().String(),
			JoinPolicy:  g.JoinPolicy().String(),
			IsGlobal:    g.IsDefault(),
			MemberCount: lg.MemberCount,
			UserRole:    memberRoleValueToStringPtr(lg.UserRole),
			CreatedAt:   g.CreatedAt().Format(timeutil.RFC3339UTC),
		})
	}

	handler.WriteJSON(w, http.StatusOK, listGroupsResponse{
		Groups:     items,
		Pagination: buildPagination(out.TotalCount, out.Page, out.TotalPages, out.Limit),
	})
}
