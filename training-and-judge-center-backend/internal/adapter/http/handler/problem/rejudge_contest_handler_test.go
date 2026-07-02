package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

func contestInGroupH(groupID string) *appProblem.ContestRejudgeInfo {
	return &appProblem.ContestRejudgeInfo{
		ID:        "c1",
		OwnerID:   "u1", // wrapAuth always authenticates as "u1"
		GroupID:   &groupID,
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now().Add(time.Hour),
	}
}

func TestRejudgeContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithRejudgeContest(
		repoReturning(publishedProblemWithJudging()),
		&mockSubmissionRejudgerH{},
		&mockContestRejudgeProviderH{contest: contestInGroupH("g1")},
	)

	r := httptest.NewRequest(http.MethodPost, "/groups/g1/contests/c1/problems/test-problem/rejudge", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	r.SetPathValue("problemSlug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.RejudgeContest).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRejudgeContest_GroupMismatch_Returns404(t *testing.T) {
	h := newHandlerWithRejudgeContest(
		repoReturning(publishedProblemWithJudging()),
		&mockSubmissionRejudgerH{},
		&mockContestRejudgeProviderH{contest: contestInGroupH("g1")},
	)

	r := authedRequest(http.MethodPost, "/groups/g2/contests/c1/problems/test-problem/rejudge", nil)
	r.SetPathValue("groupId", "g2") // real contest group is g1
	r.SetPathValue("contestId", "c1")
	r.SetPathValue("problemSlug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RejudgeContest)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRejudgeContest_Success_Returns200WithCount(t *testing.T) {
	subs := []appProblem.SubmissionRejudgeInfo{
		{ID: "s1", UserID: "u1", Language: "cpp20"},
	}
	h := newHandlerWithRejudgeContest(
		repoReturning(publishedProblemWithJudging()),
		&mockSubmissionRejudgerH{contestSubs: subs},
		&mockContestRejudgeProviderH{contest: contestInGroupH("g1"), isProblemInContest: true},
	)

	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/problems/test-problem/rejudge", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	r.SetPathValue("problemSlug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RejudgeContest)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp rejudgeContestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.SubmissionsQueued != 1 {
		t.Errorf("SubmissionsQueued = %d, want 1", resp.SubmissionsQueued)
	}
}
