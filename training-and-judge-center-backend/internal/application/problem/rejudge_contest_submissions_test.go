package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	testContestID = "dddddddd-0000-0000-0000-000000000001"
	testGroupID   = "eeeeeeee-0000-0000-0000-000000000001"
	otherGroupID  = "eeeeeeee-0000-0000-0000-000000000002"
)

// ── mock ContestRejudgeProvider ──────────────────────────────────────────────

type mockContestRejudgeProvider struct {
	contest            *ContestRejudgeInfo
	isLeadOfGroup      bool
	isProblemInContest bool
}

func (m *mockContestRejudgeProvider) GetContestForRejudge(_ context.Context, _ string) (*ContestRejudgeInfo, error) {
	return m.contest, nil
}

func (m *mockContestRejudgeProvider) IsProblemInContest(_ context.Context, _, _ string) (bool, error) {
	return m.isProblemInContest, nil
}

func (m *mockContestRejudgeProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return m.isLeadOfGroup, nil
}

func contestInGroup(groupID string) *ContestRejudgeInfo {
	return &ContestRejudgeInfo{
		ID:        testContestID,
		OwnerID:   authorID,
		GroupID:   &groupID,
		StartTime: testNow.Add(-time.Hour),
		EndTime:   testNow.Add(time.Hour),
	}
}

func TestRejudgeContestSubmissions_GroupMismatch_ReturnsNotFound(t *testing.T) {
	provider := &mockContestRejudgeProvider{contest: contestInGroup(testGroupID)}
	rejudger := &mockSubmissionRejudger{}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), rejudger, provider)

	_, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID:   testContestID,
		Slug:        testSlug,
		GroupID:     otherGroupID,
		CurrentUser: asContestant(authorID),
		Now:         testNow,
	})

	if err == nil {
		t.Fatal("expected error for group mismatch, got nil")
	}
	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != ErrCodeContestNotFound {
		t.Fatalf("expected %s, got %v", ErrCodeContestNotFound, err)
	}
}

func TestRejudgeContestSubmissions_NilGroup_ReturnsNotFound(t *testing.T) {
	contest := contestInGroup(testGroupID)
	contest.GroupID = nil
	provider := &mockContestRejudgeProvider{contest: contest}
	rejudger := &mockSubmissionRejudger{}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), rejudger, provider)

	_, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID:   testContestID,
		Slug:        testSlug,
		GroupID:     testGroupID,
		CurrentUser: asContestant(authorID),
		Now:         testNow,
	})

	if err == nil {
		t.Fatal("expected error for contest without group, got nil")
	}
	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != ErrCodeContestNotFound {
		t.Fatalf("expected %s, got %v", ErrCodeContestNotFound, err)
	}
}

func TestRejudgeContestSubmissions_GroupMatch_Success(t *testing.T) {
	sub := SubmissionRejudgeInfo{ID: "sub-001", UserID: authorID, Language: "cpp20"}
	provider := &mockContestRejudgeProvider{
		contest:            contestInGroup(testGroupID),
		isProblemInContest: true,
	}
	rejudger := &mockSubmissionRejudger{
		listContestFn: func(_ context.Context, _, _ string, _ time.Time) ([]SubmissionRejudgeInfo, error) {
			return []SubmissionRejudgeInfo{sub}, nil
		},
	}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), rejudger, provider)

	out, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID:   testContestID,
		Slug:        testSlug,
		GroupID:     testGroupID,
		CurrentUser: asContestant(authorID),
		Now:         testNow,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SubmissionsQueued != 1 {
		t.Errorf("SubmissionsQueued = %d, want 1", out.SubmissionsQueued)
	}
}
