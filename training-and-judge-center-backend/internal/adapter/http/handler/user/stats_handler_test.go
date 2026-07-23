package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithGetProfileStats(uc *appuser.GetProfileStatsUseCase) *Handler {
	return &Handler{getProfileStats: uc}
}

func TestGetStats_Unauthenticated_Returns401(t *testing.T) {
	uc := appuser.NewGetProfileStatsUseCase(nil, nil, nil, nil)
	h := newHandlerWithGetProfileStats(uc)

	req := httptest.NewRequest(http.MethodGet, "/users/me/stats", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetStats_Authenticated_EmptyData_Returns200(t *testing.T) {
	uc := appuser.NewGetProfileStatsUseCase(
		&mockRankingProvider{},
		&mockSubmissionStatsProvider{},
		&mockContestParticipationProvider{},
		&mockTopicStatsProvider{},
	)
	h := newHandlerWithGetProfileStats(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetStats),
		&domainuser.TokenClaims{UserID: "user-1", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.ProblemsSolved != 0 {
		t.Errorf("problemsSolved: want 0, got %d", resp.ProblemsSolved)
	}
	if resp.Ranking.Position != nil {
		t.Errorf("ranking.position: want null, got %v", resp.Ranking.Position)
	}
	if resp.TopicStats == nil {
		t.Error("topicStats should be an empty array, not null")
	}
}

func TestGetStats_Authenticated_PopulatedData_Returns200(t *testing.T) {
	const userID = "user-42"
	pos := 142

	rankingProvider := &mockRankingProvider{
		getRankingFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 47, &pos, 1523, nil
		},
	}
	submissionProvider := &mockSubmissionStatsProvider{
		getSubmissionCountsFn: func(_ context.Context, _ string) (int, int, error) {
			return 132, 58, nil
		},
	}
	contestProvider := &mockContestParticipationProvider{
		getContestsParticipatedCountFn: func(_ context.Context, _ string) (int, error) {
			return 9, nil
		},
	}
	topicProvider := &mockTopicStatsProvider{
		getTopicBreakdownFn: func(_ context.Context, _ string) ([]appuser.TopicStat, error) {
			return []appuser.TopicStat{{Tag: "graphs", Solved: 15}, {Tag: "dp", Solved: 12}}, nil
		},
	}

	uc := appuser.NewGetProfileStatsUseCase(rankingProvider, submissionProvider, contestProvider, topicProvider)
	h := newHandlerWithGetProfileStats(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetStats),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp statsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if resp.ProblemsSolved != 47 {
		t.Errorf("problemsSolved: want 47, got %d", resp.ProblemsSolved)
	}
	if resp.TotalSubmissions != 132 || resp.AcceptedSubmissions != 58 {
		t.Errorf("submission counts: want total=132 accepted=58, got total=%d accepted=%d", resp.TotalSubmissions, resp.AcceptedSubmissions)
	}
	if resp.ContestsParticipated != 9 {
		t.Errorf("contestsParticipated: want 9, got %d", resp.ContestsParticipated)
	}
	if resp.Ranking.Position == nil || *resp.Ranking.Position != 142 {
		t.Errorf("ranking.position: want 142, got %v", resp.Ranking.Position)
	}
	if resp.Ranking.TotalUsers != 1523 {
		t.Errorf("ranking.totalUsers: want 1523, got %d", resp.Ranking.TotalUsers)
	}
	if len(resp.TopicStats) != 2 || resp.TopicStats[0].Tag != "graphs" {
		t.Errorf("topicStats unexpected: %+v", resp.TopicStats)
	}
}
