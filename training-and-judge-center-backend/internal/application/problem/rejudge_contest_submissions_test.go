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

// HU-PRB-11: an admin who neither owns the contest nor leads its group can
// rejudge, and only the submissions of that contest are handed to the rejudger.
func TestRejudgeContestSubmissions_AdminWithoutOwnershipOrLead_Succeeds(t *testing.T) {
	var listedContest string
	provider := &mockContestRejudgeProvider{
		contest:            contestInGroup(testGroupID),
		isProblemInContest: true,
		isLeadOfGroup:      false,
	}
	rejudger := &mockSubmissionRejudger{
		listContestFn: func(_ context.Context, _, contestID string, _ time.Time) ([]SubmissionRejudgeInfo, error) {
			listedContest = contestID
			return []SubmissionRejudgeInfo{{ID: "s1"}, {ID: "s2"}}, nil
		},
	}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), rejudger, provider)

	out, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID: testContestID, Slug: testSlug, GroupID: testGroupID,
		CurrentUser: asAdmin("admin-user-id-0000-000000000001"), Now: testNow,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SubmissionsQueued != 2 || listedContest != testContestID {
		t.Errorf("queued = %d, listed contest = %q", out.SubmissionsQueued, listedContest)
	}
}

func TestRejudgeContestSubmissions_StrangerContestant_IsForbidden(t *testing.T) {
	provider := &mockContestRejudgeProvider{contest: contestInGroup(testGroupID), isProblemInContest: true}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newProblemWithJudgingUpdated()), &mockSubmissionRejudger{}, provider)

	_, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID: testContestID, Slug: testSlug, GroupID: testGroupID,
		CurrentUser: asContestant(strangerID), Now: testNow,
	})

	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected %s, got %v", ErrCodeInsufficientPermissions, err)
	}
}

func TestRejudgeContestSubmissions_AdminOnUnpublishedProblem_IsBadRequest(t *testing.T) {
	provider := &mockContestRejudgeProvider{contest: contestInGroup(testGroupID), isProblemInContest: true}
	uc := NewRejudgeContestSubmissionsUseCase(repoWith(newDraftProblemWithJudgingUpdated()), &mockSubmissionRejudger{}, provider)

	_, err := uc.Execute(context.Background(), RejudgeContestSubmissionsInput{
		ContestID: testContestID, Slug: testSlug, GroupID: testGroupID,
		CurrentUser: asAdmin("admin-user-id-0000-000000000001"), Now: testNow,
	})

	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != ErrCodeProblemNotPublished {
		t.Fatalf("expected %s, got %v", ErrCodeProblemNotPublished, err)
	}
}
