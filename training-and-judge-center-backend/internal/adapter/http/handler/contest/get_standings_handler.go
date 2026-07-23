package contest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Get contest standings (ICPC), optionally filtered by participant country, city, or institution
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId     path   string true  "Group ID"
// @Param        contestId   path   string true  "Contest ID"
// @Param        realtime    query  bool   false "Bypass freeze (lead/admin only)"
// @Param        country     query  string false "Filter by participant country (case-insensitive exact match)"
// @Param        city        query  string false "Filter by participant city (case-insensitive exact match)"
// @Param        institution query  string false "Filter by participant institution (case-insensitive exact match)"
// @Param        page        query  int    false "Page number (default 1)"
// @Param        limit       query  int    false "Items per page (default 50)"
// @Success      200 {object} getStandingsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/standings [get]
func (h *Handler) GetStandings(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
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

	realtime := q.Get("realtime") == "true"

	page := defaultPage
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}

	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			limit = n
		}
	}

	out, err := h.getStandings.Execute(r.Context(), appContest.GetStandingsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		Realtime:    realtime,
		Country:     q.Get("country"),
		City:        q.Get("city"),
		Institution: q.Get("institution"),
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toGetStandingsResponse(out))
}

func toGetStandingsResponse(out *appContest.GetStandingsOutput) getStandingsResponse {
	entries := make([]rankedEntry, len(out.Entries))
	for i, e := range out.Entries {
		probs := make(map[string]rankedProblem, len(e.Problems))
		for key, p := range e.Problems {
			rp := rankedProblem{
				Attempts: p.Attempts,
				Penalty:  p.Penalty,
				IsSolved: p.AcceptedAt != nil,
			}
			if p.AcceptedAt != nil {
				s := p.AcceptedAt.UTC().Format(time.RFC3339)
				rp.AcceptedAt = &s
			}
			probs[key] = rp
		}

		entry := rankedEntry{
			Rank:            e.Rank,
			ContestantID:    e.ContestantID,
			ParticipantType: e.ParticipantType,
			ProblemsSolved:  e.ProblemsSolved,
			TotalPenalty:    e.TotalPenalty,
			Problems:        probs,
		}
		if e.LastAcceptedAt != nil {
			s := e.LastAcceptedAt.UTC().Format(time.RFC3339)
			entry.LastAcceptedAt = &s
		}
		entries[i] = entry
	}

	totalPages := 1
	if out.Limit > 0 && out.Total > 0 {
		totalPages = (out.Total + out.Limit - 1) / out.Limit
	}

	meta := standingsMeta{
		LastUpdated:   out.Meta.LastUpdated.UTC().Format(time.RFC3339),
		IsFrozen:      out.Meta.IsFrozen,
		ContestStatus: out.Meta.ContestStatus,
	}
	if out.Meta.FrozenAt != nil {
		s := out.Meta.FrozenAt.UTC().Format(time.RFC3339)
		meta.FrozenAt = &s
	}

	return getStandingsResponse{
		Entries: entries,
		Pagination: standingsPagination{
			Page:        out.Page,
			Limit:       out.Limit,
			Total:       out.Total,
			TotalPages:  totalPages,
			HasNextPage: out.Page < totalPages,
			HasPrevPage: out.Page > 1 && totalPages > 0,
		},
		Meta: meta,
	}
}
