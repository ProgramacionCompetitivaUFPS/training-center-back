package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUpdateUC(
	repo *mockContestRepository,
	group *mockGroupProvider,
	member *mockGroupMemberProvider,
	problem *mockProblemProvider,
) *UpdateContestUseCase {
	return NewUpdateContestUseCase(repo, group, member, problem, stubOwner())
}

func validUpdateInput() UpdateContestInput {
	name := "Updated Name"
	return UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Name:        &name,
	}
}

func TestUpdateContest_SuccessUpdateName(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	out, err := uc.Execute(context.Background(), validUpdateInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", out.Name)
	}
}

func TestUpdateContest_SuccessByAdmin(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), notLead(), noProblemProvider())

	in := validUpdateInput()
	in.CurrentUser = asAdmin(testOtherID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected output, got nil")
	}
}

func TestUpdateContest_ContestNotFound(t *testing.T) {
	uc := newUpdateUC(&mockContestRepository{}, groupFound(), isLead(), noProblemProvider())

	_, err := uc.Execute(context.Background(), validUpdateInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND, got %v", err)
	}
}

func TestUpdateContest_ContestBelongsToDifferentGroup(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	in := validUpdateInput()
	in.GroupID = "different-group-id"

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND when group mismatch, got %v", err)
	}
}

func TestUpdateContest_ForbiddenIfNotLead(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), notLead(), noProblemProvider())

	_, err := uc.Execute(context.Background(), validUpdateInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestUpdateContest_LockedContestForbidsNonOwner(t *testing.T) {
	contest := newTestContest(testOtherID) // owned by testOtherID
	locked := true
	contest.SetLocked(locked, testNow)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	// testCallerID is lead but NOT the owner (testOtherID is owner)
	_, err := uc.Execute(context.Background(), validUpdateInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestLocked {
		t.Errorf("expected CONTEST_LOCKED, got %v", err)
	}
}

func TestUpdateContest_OwnerCanUnlockOwnContest(t *testing.T) {
	contest := newTestContest(testCallerID) // owned by testCallerID
	contest.SetLocked(true, testNow)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	locked := false
	out, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Locked:      &locked,
	})

	if err != nil {
		t.Fatalf("expected owner to unlock, got error: %v", err)
	}
	if out.Locked {
		t.Error("expected contest to be unlocked")
	}
}

func TestUpdateContest_AdminCanUnlockAnyContest(t *testing.T) {
	contest := newTestContest(testOtherID) // owned by someone else
	contest.SetLocked(true, testNow)
	uc := newUpdateUC(repoWith(contest), groupFound(), notLead(), noProblemProvider())

	locked := false
	out, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asAdmin(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Locked:      &locked,
	})

	if err != nil {
		t.Fatalf("expected admin to unlock, got error: %v", err)
	}
	if out.Locked {
		t.Error("expected contest to be unlocked after admin unlock")
	}
}

func TestUpdateContest_NonOwnerNonAdminCannotLock(t *testing.T) {
	contest := newTestContest(testOtherID) // owned by testOtherID
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	locked := true
	_, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID), // not owner, not admin
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Locked:      &locked,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeOnlyOwnerOrAdmin {
		t.Errorf("expected ONLY_OWNER_OR_ADMIN_CAN_UNLOCK, got %v", err)
	}
}

func TestUpdateContest_NoFieldsToUpdate(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	_, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		// no fields
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNoFieldsToUpdate {
		t.Errorf("expected NO_FIELDS_TO_UPDATE, got %v", err)
	}
}

func TestUpdateContest_StartTimeInPast(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	past := time.Now().Add(-time.Hour) // genuinely in the past
	_, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		StartTime:   &past,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "START_TIME_IN_PAST" {
		t.Errorf("expected START_TIME_IN_PAST, got %v", err)
	}
}

func TestUpdateContest_EndTimeInPast(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	past := time.Now().Add(-time.Minute) // genuinely in the past
	_, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		EndTime:     &past,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "END_TIME_IN_PAST" {
		t.Errorf("expected END_TIME_IN_PAST, got %v", err)
	}
}

func TestUpdateContest_ReplaceProblems(t *testing.T) {
	contest := newTestContest(testCallerID)
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), noProblemProvider())

	problems := []ProblemOrderInput{
		{Slug: "problem-a", Order: 1},
		{Slug: "problem-b", Order: 2},
	}
	out, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Problems:    &problems,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) != 2 {
		t.Errorf("expected 2 problems, got %d", len(out.Problems))
	}
}

func TestUpdateContest_ProblemAccessDeniedOnUpdate(t *testing.T) {
	contest := newTestContest(testCallerID)
	pp := &mockProblemProvider{
		findBySlugsFn: func(_ context.Context, slugs []string, _ string, _ bool) (map[string]*ProblemInfo, error) {
			result := make(map[string]*ProblemInfo)
			for _, s := range slugs {
				result[s] = &ProblemInfo{ID: testProblemID, Slug: s, Title: s, IsPublished: true, CanAdd: false}
			}
			return result, nil
		},
	}
	uc := newUpdateUC(repoWith(contest), groupFound(), isLead(), pp)

	problems := []ProblemOrderInput{{Slug: "private-problem", Order: 1}}
	_, err := uc.Execute(context.Background(), UpdateContestInput{
		CurrentUser: asCoach(testCallerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Problems:    &problems,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemAccessDenied {
		t.Errorf("expected PROBLEM_ACCESS_DENIED, got %v", err)
	}
}
