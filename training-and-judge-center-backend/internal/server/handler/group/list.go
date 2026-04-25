package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	page, err := parseIntParam(q.Get("page"), 1)
	if err != nil {
		writeBadPagination(w, "page", "page must be a positive integer")
		return
	}
	limit, err := parseIntParam(q.Get("limit"), appGroup.DefaultPageLimit)
	if err != nil {
		writeBadPagination(w, "limit", "limit must be an integer")
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

	out, err := h.listUC.Execute(r.Context(), in)
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
			UserRole:    memberRoleToStringPtr(lg.UserRole),
			CreatedAt:   g.CreatedAt().Format(timestampFormat),
		})
	}

	handler.WriteJSON(w, http.StatusOK, listGroupsResponse{
		Groups:     items,
		Pagination: buildPagination(out.TotalCount, out.Page, out.TotalPages, out.Limit),
	})
}

func writeBadPagination(w http.ResponseWriter, field, msg string) {
	handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
		Code:    apperror.ErrCodeValidationError,
		Message: "Invalid request parameters",
		Details: []apperror.FieldError{{Field: field, Message: msg}},
	})
}
