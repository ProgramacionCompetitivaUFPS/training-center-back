package contest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      List contest submissions
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId      path   string true  "Group ID"
// @Param        contestId    path   string true  "Contest ID"
// @Param        problemSlug  query  string false "Filter by problem slug"
// @Param        nickname     query  string false "Filter by participant nickname"
// @Param        phase        query  string false "Filter by phase: competition, postcompetition (default: all)"
// @Param        page         query  int    false "Page number (default 1)"
// @Param        limit        query  int    false "Items per page (default 50, max 100)"
// @Success      200 {object} listSubmissionsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/submissions [get]
func (h *Handler) ListContestSubmissions(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()

	var problemSlug *string
	if v := q.Get("problemSlug"); v != "" {
		problemSlug = &v
	}
	var nickname *string
	if v := q.Get("nickname"); v != "" {
		nickname = &v
	}

	phase := q.Get("phase")
	if phase != "" && phase != "competition" && phase != "postcompetition" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "phase must be 'competition' or 'postcompetition'"))
		return
	}

	page := defaultPage
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}
	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}

	out, err := h.listContestSubmissions.Execute(r.Context(), appContest.ListContestSubmissionsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		ProblemSlug: problemSlug,
		Nickname:    nickname,
		Phase:       phase,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toListSubmissionsResponse(out))
}

func toListSubmissionsResponse(out *appContest.ListContestSubmissionsOutput) listSubmissionsResponse {
	items := make([]submissionItem, len(out.Submissions))
	for i, s := range out.Submissions {
		item := submissionItem{
			ID: s.ID,
			Problem: submissionProblem{
				Slug:  s.Problem.Slug,
				Title: s.Problem.Title,
				Order: s.Problem.Order,
			},
			SubmittedBy: submissionSubmitter{Nickname: s.SubmittedBy.Nickname},
			Language:    s.Language,
			Status:      s.Status,
			SubmittedAt: s.SubmittedAt.UTC().Format(time.RFC3339),
			TimeMs:      s.TimeMs,
			MemoryKb:    s.MemoryKb,
			Phase:       s.Phase,
		}
		if s.JudgedAt != nil {
			v := s.JudgedAt.UTC().Format(time.RFC3339)
			item.JudgedAt = &v
		}
		items[i] = item
	}

	totalPages := appshared.CalcTotalPages(out.Total, out.Limit)

	meta := submissionsContestMeta{
		ID:       out.Meta.ID,
		Name:     out.Meta.Name,
		Status:   out.Meta.Status,
		InFreeze: out.Meta.InFreeze,
	}
	if out.Meta.FreezeTime != nil {
		v := out.Meta.FreezeTime.UTC().Format(time.RFC3339)
		meta.FreezeTime = &v
	}

	return listSubmissionsResponse{
		Contest:     meta,
		Submissions: items,
		Pagination: standingsPagination{
			Page:       out.Page,
			Limit:      out.Limit,
			Total:      out.Total,
			TotalPages: totalPages,
		},
	}
}
