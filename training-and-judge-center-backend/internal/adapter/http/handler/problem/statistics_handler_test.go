package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func defaultStatisticsUC(repo domainProblem.Repository, provider appProblem.StatisticsProvider) *appProblem.GetProblemStatisticsUseCase {
	return appProblem.NewGetProblemStatisticsUseCase(repo, provider)
}

func TestGetStatistics_Unauthenticated_Returns401(t *testing.T) {
	uc := defaultStatisticsUC(repoReturning(publishedProblem()), &mockStatsProvider{stats: &appProblem.ProblemStats{}})
	h := newHandlerWithStatistics(uc)

	r := httptest.NewRequest(http.MethodGet, "/problems/p/test-problem/statistics", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetStatistics).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetStatistics_NotFound_Returns404(t *testing.T) {
	uc := defaultStatisticsUC(&mockProblemRepo{}, &mockStatsProvider{})
	h := newHandlerWithStatistics(uc)

	r := authedRequest(http.MethodGet, "/problems/p/missing/statistics", nil)
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStatistics)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStatistics_Draft_Returns403(t *testing.T) {
	uc := defaultStatisticsUC(repoReturning(draftProblem()), &mockStatsProvider{})
	h := newHandlerWithStatistics(uc)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/statistics", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStatistics)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var errResp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if errResp.Code != appProblem.ErrCodeProblemNotPublished {
		t.Errorf("expected code %s, got %s", appProblem.ErrCodeProblemNotPublished, errResp.Code)
	}
}

func TestGetStatistics_NoSubmissions_Returns200WithMessage(t *testing.T) {
	uc := defaultStatisticsUC(
		repoReturning(publishedProblem()),
		&mockStatsProvider{stats: &appProblem.ProblemStats{TotalSubmissions: 0}},
	)
	h := newHandlerWithStatistics(uc)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/statistics", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStatistics)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp noSubmissionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.TotalSubmissions != 0 {
		t.Errorf("expected totalSubmissions=0, got %d", resp.TotalSubmissions)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestGetStatistics_WithSubmissions_Returns200WithFullStats(t *testing.T) {
	stats := &appProblem.ProblemStats{
		TotalSubmissions: 10,
		UniqueAttempted:  5,
		UniqueSolved:     3,
		ByLanguage: []appProblem.LanguageStat{
			{Language: "cpp20", UsersAccepted: 3, UsersAttempted: 5},
		},
		ByVerdict: []appProblem.VerdictStat{
			{Verdict: "ACCEPTED", Count: 4},
			{Verdict: "WRONG_ANSWER", Count: 6},
		},
	}
	uc := defaultStatisticsUC(repoReturning(publishedProblem()), &mockStatsProvider{stats: stats})
	h := newHandlerWithStatistics(uc)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/statistics", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStatistics)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp statisticsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.TotalSubmissions != 10 {
		t.Errorf("expected totalSubmissions=10, got %d", resp.TotalSubmissions)
	}
	if resp.UniqueUsers.Attempted != 5 {
		t.Errorf("expected uniqueUsers.attempted=5, got %d", resp.UniqueUsers.Attempted)
	}
	if resp.UniqueUsers.Solved != 3 {
		t.Errorf("expected uniqueUsers.solved=3, got %d", resp.UniqueUsers.Solved)
	}
	if len(resp.AcceptanceRateByLanguage) != 1 {
		t.Errorf("expected 1 language stat, got %d", len(resp.AcceptanceRateByLanguage))
	}
	if len(resp.VerdictDistribution) != 2 {
		t.Errorf("expected 2 verdict stats, got %d", len(resp.VerdictDistribution))
	}
}
