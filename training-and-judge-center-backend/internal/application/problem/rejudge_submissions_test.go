package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mock SubmissionRejudger ──────────────────────────────────────────────────

type mockSubmissionRejudger struct {
	listFn      func(ctx context.Context, problemID string, before time.Time) ([]SubmissionRejudgeInfo, error)
	rejudgeOneFn func(ctx context.Context, info SubmissionRejudgeInfo, problemID string, now time.Time) error
	rejudged    []string // records IDs passed to RejudgeOne
}

func (m *mockSubmissionRejudger) ListByProblemBefore(ctx context.Context, problemID string, before time.Time) ([]SubmissionRejudgeInfo, error) {
	if m.listFn != nil {
		return m.listFn(ctx, problemID, before)
	}
	return nil, nil
}

func (m *mockSubmissionRejudger) RejudgeOne(ctx context.Context, info SubmissionRejudgeInfo, problemID string, now time.Time) error {
	m.rejudged = append(m.rejudged, info.ID)
	if m.rejudgeOneFn != nil {
		return m.rejudgeOneFn(ctx, info, problemID, now)
	}
	return nil
}

// ── fixtures ─────────────────────────────────────────────────────────────────

var judgingUpdated = time.Date(2024, 1, 14, 8, 0, 0, 0, time.UTC)

func newProblemWithJudgingUpdated() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(authorID),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, &judgingUpdated,
		testNow, testNow,
	)
}

func newProblemNoJudgingUpdated() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(authorID),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRejudgeSubmissions_Execute(t *testing.T) {
	sub1 := SubmissionRejudgeInfo{ID: "sub-001", UserID: authorID, Language: "cpp20"}
	sub2 := SubmissionRejudgeInfo{ID: "sub-002", UserID: authorID, Language: "java17"}

	tests := []struct {
		name        string
		problem     *domainProblem.Problem
		user        func(id string) appshared.CurrentUser
		userID      string
		submissions []SubmissionRejudgeInfo
		rejudgeErr  error
		wantQueued  int
		wantErrCode string
	}{
		{
			name:        "author rejudges — 2 submissions queued",
			problem:     newProblemWithJudgingUpdated(),
			user:        asContestant,
			userID:      authorID,
			submissions: []SubmissionRejudgeInfo{sub1, sub2},
			wantQueued:  2,
		},
		{
			name:        "admin rejudges — ok",
			problem:     newProblemWithJudgingUpdated(),
			user:        asAdmin,
			userID:      "admin-user-id-0000-000000000001",
			submissions: []SubmissionRejudgeInfo{sub1},
			wantQueued:  1,
		},
		{
			name:        "no judgingUpdatedAt — returns 0 queued without calling rejudger",
			problem:     newProblemNoJudgingUpdated(),
			user:        asContestant,
			userID:      authorID,
			submissions: nil,
			wantQueued:  0,
		},
		{
			name:        "stranger forbidden",
			problem:     newProblemWithJudgingUpdated(),
			user:        asContestant,
			userID:      strangerID,
			wantErrCode: ErrCodeInsufficientPermissions,
		},
		{
			name:        "all submissions fail to enqueue — none queued",
			problem:     newProblemWithJudgingUpdated(),
			user:        asContestant,
			userID:      authorID,
			submissions: []SubmissionRejudgeInfo{sub1, sub2},
			rejudgeErr:  errors.New("queue down"),
			wantQueued:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rejudger := &mockSubmissionRejudger{
				listFn: func(_ context.Context, _ string, _ time.Time) ([]SubmissionRejudgeInfo, error) {
					return tc.submissions, nil
				},
				rejudgeOneFn: func(_ context.Context, _ SubmissionRejudgeInfo, _ string, _ time.Time) error {
					return tc.rejudgeErr
				},
			}

			uc := NewRejudgeSubmissionsUseCase(repoWith(tc.problem), rejudger)
			currentUser := tc.user(tc.userID)

			out, err := uc.Execute(context.Background(), RejudgeSubmissionsInput{
				Slug:        testSlug,
				CurrentUser: currentUser,
				Now:         testNow,
			})

			if tc.wantErrCode != "" {
				if err == nil {
					t.Fatalf("expected error %s, got nil", tc.wantErrCode)
				}
				var ae *apperror.AppError
				if !errors.As(err, &ae) || ae.Code != tc.wantErrCode {
					t.Fatalf("expected error code %s, got %v", tc.wantErrCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.SubmissionsQueued != tc.wantQueued {
				t.Errorf("SubmissionsQueued = %d, want %d", out.SubmissionsQueued, tc.wantQueued)
			}
			if out.ProblemSlug != testSlug {
				t.Errorf("ProblemSlug = %q, want %q", out.ProblemSlug, testSlug)
			}
		})
	}
}

func TestRejudgeSubmissions_PartialFailure_ContinuesAndCountsSuccesses(t *testing.T) {
	sub1 := SubmissionRejudgeInfo{ID: "sub-001", UserID: authorID, Language: "cpp20"}
	sub2 := SubmissionRejudgeInfo{ID: "sub-002", UserID: authorID, Language: "java17"}

	callCount := 0
	rejudger := &mockSubmissionRejudger{
		listFn: func(_ context.Context, _ string, _ time.Time) ([]SubmissionRejudgeInfo, error) {
			return []SubmissionRejudgeInfo{sub1, sub2}, nil
		},
		rejudgeOneFn: func(_ context.Context, _ SubmissionRejudgeInfo, _ string, _ time.Time) error {
			callCount++
			if callCount == 1 {
				return errors.New("queue down")
			}
			return nil
		},
	}

	uc := NewRejudgeSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), rejudger)
	out, err := uc.Execute(context.Background(), RejudgeSubmissionsInput{
		Slug:        testSlug,
		CurrentUser: asContestant(authorID),
		Now:         testNow,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SubmissionsQueued != 1 {
		t.Errorf("SubmissionsQueued = %d, want 1 (first fails, second succeeds)", out.SubmissionsQueued)
	}
	if callCount != 2 {
		t.Errorf("RejudgeOne called %d times, want 2 (loop must continue on error)", callCount)
	}
}
