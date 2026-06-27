package contest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	defaultPage  = 1
	defaultLimit = 20
)

// @Summary      List contests
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path   string true  "Group ID"
// @Param        status  query  string false "Filter by status (SCHEDULED, ACTIVE, FINISHED)"
// @Param        sortBy  query  string false "Sort field (name, startTime, createdAt)"
// @Param        order   query  string false "Sort order (asc, desc)"
// @Param        page    query  int    false "Page number (default 1)"
// @Param        limit   query  int    false "Items per page (default 20, max 100)"
// @Success      200 {object} listContestsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing groupId"))
		return
	}

	q := r.URL.Query()

	var statusFilter *string
	if s := q.Get("status"); s != "" {
		statusFilter = &s
	}

	page := defaultPage
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}

	limit := defaultLimit
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			limit = n
		}
	}

	out, err := h.listContests.Execute(r.Context(), appContest.ListContestsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		Status:      statusFilter,
		SortBy:      q.Get("sortBy"),
		Order:       q.Get("order"),
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toListContestsResponse(out))
}

func toListContestsResponse(out *appContest.ListContestsOutput) listContestsResponse {
	items := make([]contestListItem, len(out.Items))
	for i, item := range out.Items {
		items[i] = contestListItem{
			ID:                item.ID,
			Name:              item.Name,
			Description:       item.Description,
			StartTime:         item.StartTime.UTC().Format(time.RFC3339),
			EndTime:           item.EndTime.UTC().Format(time.RFC3339),
			Duration:          item.Duration,
			Status:            item.Status,
			Penalty:           item.Penalty,
			FreezeMinutes:     item.FreezeMinutes,
			EnablePostContest: item.EnablePostContest,
			ParticipantCount:  item.ParticipantCount,
			IsRegistered:      item.IsRegistered,
			ProblemCount:      item.ProblemCount,
		}
	}
	return listContestsResponse{
		Items: items,
		Pagination: pagination{
			Page:        out.Pagination.Page,
			Limit:       out.Pagination.Limit,
			Total:       out.Pagination.Total,
			TotalPages:  out.Pagination.TotalPages,
			HasNextPage: out.Pagination.Page < out.Pagination.TotalPages,
			HasPrevPage: out.Pagination.Page > 1 && out.Pagination.TotalPages > 0,
		},
	}
}
