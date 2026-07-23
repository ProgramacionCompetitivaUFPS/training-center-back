package group

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      List group members
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        page  query int    false "Page number"
// @Param        limit query int    false "Page size"
// @Param        role  query string false "Filter by role (LEAD, MEMBER)"
// @Success      200 {object} listMembersResp
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/members [get]
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	page, limit, ok := parsePaginationParams(r.Context(), q, w)
	if !ok {
		return
	}

	var roleFilter *string
	if rv := q.Get("role"); rv != "" {
		roleFilter = &rv
	}

	out, err := h.listMembers.Execute(r.Context(), appGroup.ListMembersInput{
		GroupID:     r.PathValue("groupId"),
		Page:        page,
		Limit:       limit,
		Role:        roleFilter,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]memberListItemResp, 0, len(out.Members))
	for _, m := range out.Members {
		items = append(items, memberListItemResp{
			UserID:   m.UserID,
			Nickname: m.Nickname,
			Name:     m.Name,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
		})
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listMembersResp{
		Members:    items,
		Pagination: buildPagination(out.TotalCount, page, out.TotalPages, limit),
	})
}
