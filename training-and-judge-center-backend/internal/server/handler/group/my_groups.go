package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

// @Summary      List my groups
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Param        role query string false "Filter by role"
// @Success      200 {object} listMyGroupsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/groups [get]
func (h *Handler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	page, limit, ok := parsePaginationParams(q, w)
	if !ok {
		return
	}

	in := appGroup.ListMyGroupsInput{
		CurrentUser: *currentUser,
		Role:        stringPtrOrNil(q.Get("role")),
		Search:      q.Get("search"),
		SortBy:      q.Get("sortBy"),
		Order:       q.Get("order"),
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listMyGroups.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	items := make([]myGroupItemResp, 0, len(out.Groups))
	for _, mg := range out.Groups {
		g := mg.Group
		items = append(items, myGroupItemResp{
			ID:          g.ID(),
			Name:        g.Name().String(),
			Description: g.Description(),
			Visibility:  g.Visibility().String(),
			JoinPolicy:  g.JoinPolicy().String(),
			IsGlobal:    g.IsDefault(),
			MyRole:      string(mg.MyRole),
			JoinedAt:    mg.JoinedAt.Format(timeutil.RFC3339UTC),
			MemberCount: mg.MemberCount,
			CreatedAt:   g.CreatedAt().Format(timeutil.RFC3339UTC),
		})
	}

	handler.WriteJSON(w, http.StatusOK, listMyGroupsResponse{
		Groups:     items,
		Pagination: buildPagination(out.TotalCount, out.Page, out.TotalPages, out.Limit),
	})
}
