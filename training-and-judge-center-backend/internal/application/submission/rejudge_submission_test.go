package submission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockJudgingProvider struct{ published bool }

func (m *mockJudgingProvider) GetJudgingUpdatedAt(context.Context, string) (*time.Time, error) {
	t := testNow.Add(time.Hour)
	return &t, nil
}

func (m *mockJudgingProvider) IsPublished(_ context.Context, problemID string) (bool, error) {
	return m.published && problemID != "", nil
}

type mockSingleRejudger struct{ calls []string }

func (m *mockSingleRejudger) RejudgeByID(_ context.Context, id, _, _ string, _ *string, _ string, _ time.Time) error {
	m.calls = append(m.calls, id)
	return nil
}

func finishedSubmission(problemID string) *domainsubmission.Submission {
	return domainsubmission.RestoreSubmission(
		testSubmissionID, problemID, shared.RestoreUserID(testUserID), nil, nil,
		domainsubmission.RestoreLanguage("cpp20"), "g++",
		domainsubmission.RestoreStatus("WRONG_ANSWER"),
		domainsubmission.RestoreVisibility("PRIVATE"),
		testSourcePath, "hash", 100, testNow, nil, nil, nil, nil, "", "",
	)
}

func newRejudgeUseCase(sub *domainsubmission.Submission, published bool) (*RejudgeSubmissionUseCase, *mockSingleRejudger) {
	rejudger := &mockSingleRejudger{}
	repo := &mockSubmissionRepo{findByIDFn: func(string) (*domainsubmission.Submission, error) { return sub, nil }}
	return NewRejudgeSubmissionUseCase(repo, &mockJudgingProvider{published: published}, nil, rejudger), rejudger
}

func TestRejudgeSubmission_PublishedProblem_IsRejudged(t *testing.T) {
	uc, rejudger := newRejudgeUseCase(finishedSubmission(testProblemID), true)

	if _, err := uc.Execute(context.Background(), RejudgeSubmissionInput{SubmissionID: testSubmissionID, CurrentUser: asAdmin("admin-1"), Now: testNow}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rejudger.calls) != 1 {
		t.Errorf("rejudger calls = %d, want 1", len(rejudger.calls))
	}
}

func TestRejudgeSubmission_UnpublishedProblem_IsRejected(t *testing.T) {
	uc, rejudger := newRejudgeUseCase(finishedSubmission(testProblemID), false)

	_, err := uc.Execute(context.Background(), RejudgeSubmissionInput{SubmissionID: testSubmissionID, CurrentUser: asAdmin("admin-1"), Now: testNow})

	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != domainsubmission.ErrCodeProblemNotPublished {
		t.Fatalf("expected %s, got %v", domainsubmission.ErrCodeProblemNotPublished, err)
	}
	if len(rejudger.calls) != 0 {
		t.Error("nothing should be queued")
	}
}

// Deleting a problem leaves its submissions without a problem_id.
func TestRejudgeSubmission_DeletedProblem_ReportsNotFound(t *testing.T) {
	uc, rejudger := newRejudgeUseCase(finishedSubmission(""), true)

	_, err := uc.Execute(context.Background(), RejudgeSubmissionInput{SubmissionID: testSubmissionID, CurrentUser: asAdmin("admin-1"), Now: testNow})

	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != domainsubmission.ErrCodeProblemNotFound {
		t.Fatalf("expected %s, got %v", domainsubmission.ErrCodeProblemNotFound, err)
	}
	if len(rejudger.calls) != 0 {
		t.Error("nothing should be queued")
	}
}
