package problem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// ── mock SubmissionRejudger ──────────────────────────────────────────────────

type mockSubmissionRejudgerH struct {
	subs    []appProblem.SubmissionRejudgeInfo
	listErr error
}

func (m *mockSubmissionRejudgerH) ListByProblemBefore(_ context.Context, _ string, _ time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	return m.subs, m.listErr
}

func (m *mockSubmissionRejudgerH) ListByProblemAndContestBefore(_ context.Context, _, _ string, _ time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	return nil, nil
}

func (m *mockSubmissionRejudgerH) RejudgeBatch(_ context.Context, subs []appProblem.SubmissionRejudgeInfo, _ string, _ time.Time) (int, error) {
	return len(subs), nil
}

// ── fixtures ─────────────────────────────────────────────────────────────────

func publishedProblemWithJudging() *domainProblem.Problem {
	ts := time.Now().Add(-time.Hour)
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID("u1"),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, &ts,
		testNow, testNow,
	)
}

func publishedProblemWithJudgingByOtherAuthor() *domainProblem.Problem {
	ts := time.Now().Add(-time.Hour)
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID("u2"), // wrapAuth always creates "u1"
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, &ts,
		testNow, testNow,
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRejudge_Forbidden_Returns403(t *testing.T) {
	h := newHandlerWithRejudge(repoReturning(publishedProblemWithJudgingByOtherAuthor()), &mockSubmissionRejudgerH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/rejudge", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Rejudge)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRejudge_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithRejudge(repoReturning(publishedProblemWithJudging()), &mockSubmissionRejudgerH{})

	r := httptest.NewRequest(http.MethodPost, "/problems/p/test-problem/rejudge", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Rejudge).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRejudge_ProblemNotFound_Returns404(t *testing.T) {
	h := newHandlerWithRejudge(&mockProblemRepo{}, &mockSubmissionRejudgerH{})

	r := authedRequest(http.MethodPost, "/problems/p/missing/rejudge", nil)
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Rejudge)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRejudge_Success_Returns200WithCount(t *testing.T) {
	subs := []appProblem.SubmissionRejudgeInfo{
		{ID: "s1", UserID: "u1", Language: "cpp20"},
		{ID: "s2", UserID: "u2", Language: "java17"},
	}
	h := newHandlerWithRejudge(
		repoReturning(publishedProblemWithJudging()),
		&mockSubmissionRejudgerH{subs: subs},
	)

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/rejudge", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Rejudge)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp rejudgeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.SubmissionsQueued != 2 {
		t.Errorf("SubmissionsQueued = %d, want 2", resp.SubmissionsQueued)
	}
	if resp.ProblemSlug != "test-problem" {
		t.Errorf("ProblemSlug = %q, want %q", resp.ProblemSlug, "test-problem")
	}
}

func TestRejudge_NoJudgingUpdatedAt_Returns400(t *testing.T) {
	h := newHandlerWithRejudge(
		repoReturning(publishedProblem()),
		&mockSubmissionRejudgerH{},
	)

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/rejudge", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Rejudge)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
