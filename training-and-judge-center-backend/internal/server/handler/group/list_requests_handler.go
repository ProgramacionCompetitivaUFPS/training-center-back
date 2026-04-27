package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
)

func (h *Handler) ListJoinRequests(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")
	q := r.URL.Query()

	page, limit, ok := parsePaginationParams(q, w)
	if !ok {
		return
	}

	out, err := h.listRequests.Execute(r.Context(), appGroup.ListJoinRequestsInput{
		GroupID:     groupID,
		Status:      q.Get("status"),
		Page:        page,
		Limit:       limit,
		CurrentUser: *caller,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	items := make([]joinRequestResp, 0, len(out.Requests))
	for _, detail := range out.Requests {
		items = append(items, buildJoinRequestResp(detail.Request, detail.Display))
	}

	handler.WriteJSON(w, http.StatusOK, listRequestsResponse{
		Requests:   items,
		Pagination: buildPagination(out.Total, 1, out.TotalPages, limit),
	})
}
