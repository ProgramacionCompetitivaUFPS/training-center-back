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

	country := q.Get("country")
	city := q.Get("city")
	institution := q.Get("institution")

	out, err := h.getStandings.Execute(r.Context(), appContest.GetStandingsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		Realtime:    realtime,
		Country:     country,
		City:        city,
		Institution: institution,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toGetStandingsResponse(out, country, city, institution))
}

func toGetStandingsResponse(out *appContest.GetStandingsOutput, country, city, institution string) getStandingsResponse {
	standings := make([]standingEntry, len(out.Entries))
	for i, e := range out.Entries {
		standings[i] = standingEntry{
			Rank:           e.Rank,
			Participant:    toStandingParticipant(e.Participant),
			ProblemsSolved: e.ProblemsSolved,
			TotalPenalty:   e.TotalPenalty,
			Problems:       toStandingProblemResults(e.Problems, out.Problems, out.Contest.StartTime),
		}
	}

	problems := make([]standingsProblemHeader, len(out.Problems))
	for i, p := range out.Problems {
		problems[i] = standingsProblemHeader{Position: p.Position, Slug: p.Slug, Title: p.Title}
	}

	totalPages := 1
	if out.Limit > 0 && out.Total > 0 {
		totalPages = (out.Total + out.Limit - 1) / out.Limit
	}

	var frozenAt *string
	if out.Meta.FrozenAt != nil {
		s := out.Meta.FrozenAt.UTC().Format(time.RFC3339)
		frozenAt = &s
	}

	var freezeMinutes *int
	if out.Contest.FreezeMinutes > 0 {
		fm := out.Contest.FreezeMinutes
		freezeMinutes = &fm
	}

	return getStandingsResponse{
		Contest: standingsContestDisplay{
			ID:            out.Contest.ID,
			Name:          out.Contest.Name,
			Status:        out.Contest.Status,
			StartTime:     out.Contest.StartTime.UTC().Format(time.RFC3339),
			EndTime:       out.Contest.EndTime.UTC().Format(time.RFC3339),
			Penalty:       out.Contest.Penalty,
			FreezeMinutes: freezeMinutes,
			IsFrozen:      out.Contest.IsFrozen,
			FrozenAt:      frozenAt,
		},
		Problems:  problems,
		Standings: standings,
		Pagination: standingsPagination{
			Page:        out.Page,
			Limit:       out.Limit,
			Total:       out.Total,
			TotalPages:  totalPages,
			HasNextPage: out.Page < totalPages,
			HasPrevPage: out.Page > 1 && totalPages > 0,
		},
		Filters: standingsFilters{
			Country:       nilIfEmptyString(country),
			City:          nilIfEmptyString(city),
			Institution:   nilIfEmptyString(institution),
			FilteredTotal: out.Total,
		},
	}
}

func toStandingParticipant(p appContest.StandingParticipantDisplay) standingParticipant {
	return standingParticipant{
		ID:          p.ID,
		Type:        p.Type,
		DisplayName: p.DisplayName,
		Nickname:    p.Nickname,
		Name:        p.Name,
		Members:     p.Members,
		Country:     p.Country,
		City:        p.City,
		Institution: p.Institution,
	}
}

// toStandingProblemResults reshapes the ranked-entry's problem map (keyed by
// problem ID) into the ordered-by-position array the API contract exposes,
// filling in NOT_ATTEMPTED for contest problems the entry never touched.
func toStandingProblemResults(
	entryProblems map[string]appContest.RankedProblem,
	contestProblems []appContest.StandingsProblemDisplay,
	contestStart time.Time,
) []standingProblemResult {
	results := make([]standingProblemResult, len(contestProblems))
	for i, cp := range contestProblems {
		rp, attempted := entryProblems[cp.ID]
		result := standingProblemResult{Position: cp.Position, Status: "NOT_ATTEMPTED"}
		if attempted {
			result.Attempts = rp.Attempts
			result.Penalty = rp.Penalty
			if rp.AcceptedAt != nil {
				result.Status = "ACCEPTED"
				minutes := int(rp.AcceptedAt.Sub(contestStart).Minutes())
				result.Time = &minutes
			} else if rp.Attempts > 0 {
				result.Status = "WRONG_ANSWER"
			}
		}
		results[i] = result
	}
	return results
}

func nilIfEmptyString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
