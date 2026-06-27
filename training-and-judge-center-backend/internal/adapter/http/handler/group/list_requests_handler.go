package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      List join requests for a group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        status query string false "Filter by status (PENDING, APPROVED, REJECTED)"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} listRequestsResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/requests [get]
func (h *Handler) ListJoinRequests(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")
	q := r.URL.Query()

	page, limit, ok := parsePaginationParams(r.Context(), q, w)
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
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]joinRequestResp, 0, len(out.Requests))
	for _, detail := range out.Requests {
		items = append(items, buildJoinRequestResp(detail.Request, detail.Display))
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listRequestsResponse{
		Requests:   items,
		Pagination: buildPagination(out.Total, page, out.TotalPages, limit),
	})
}
