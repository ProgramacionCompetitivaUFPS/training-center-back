package contest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      List contest registrations
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId   path  string true  "Group ID"
// @Param        contestId path  string true  "Contest ID"
// @Param        page      query int    false "Page (default 1)"
// @Param        limit     query int    false "Limit (default 50, max 100)"
// @Success      200 {object} listRegistrationsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/registrations [get]
func (h *Handler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v >= 1 {
			page = v
		}
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v >= 1 && v <= 100 {
			limit = v
		}
	}

	out, err := h.listContestRegistrations.Execute(r.Context(), appContest.ListContestRegistrationsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]registrationItem, len(out.Registrations))
	for i, e := range out.Registrations {
		items[i] = registrationItem{
			Nickname:     e.Nickname,
			RegisteredAt: e.RegisteredAt.UTC().Format(time.RFC3339),
		}
	}

	totalPages := 0
	if out.Limit > 0 && out.Total > 0 {
		totalPages = (out.Total + out.Limit - 1) / out.Limit
	}
	hasMore := out.Page < totalPages

	handler.WriteJSON(r.Context(), w, http.StatusOK, listRegistrationsResponse{
		Registrations: items,
		Pagination: registrationsPagination{
			Page:       out.Page,
			Limit:      out.Limit,
			Total:      out.Total,
			TotalPages: totalPages,
			HasMore:    hasMore,
		},
	})
}
