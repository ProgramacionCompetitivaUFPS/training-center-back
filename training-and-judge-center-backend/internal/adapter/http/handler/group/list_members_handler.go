package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
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

	totalPages := out.TotalCount / limit
	if out.TotalCount%limit != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]memberListItemResp, 0, len(out.Members))
	for _, m := range out.Members {
		items = append(items, memberListItemResp{
			UserID:   m.UserID,
			Nickname: m.Nickname,
			Name:     m.Name,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.Format(timeutil.RFC3339UTC),
		})
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listMembersResp{
		Members:    items,
		Pagination: buildPagination(out.TotalCount, page, totalPages, limit),
	})
}
