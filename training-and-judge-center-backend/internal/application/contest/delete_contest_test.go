package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newDeleteContestUseCase(contest *domainContest.Contest, memberProvider *mockGroupMemberProvider) *DeleteContestUseCase {
	return NewDeleteContestUseCase(
		repoWith(contest),
		memberProvider,
		&mockStandingsCache{},
	)
}

func scheduledContestFixture() *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Scheduled Contest"),
		nil,
		time.Now().Add(2*time.Hour),
		time.Now().Add(4*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func TestDeleteContest_NotLead_Returns403(t *testing.T) {
	uc := newDeleteContestUseCase(activeContest(), isMemberNotLead())

	err := uc.Execute(context.Background(), DeleteContestInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected ErrCodeInsufficientPermissions, got %v", err)
	}
}

func TestDeleteContest_ContestNotFound_Returns404(t *testing.T) {
	uc := NewDeleteContestUseCase(
		&mockContestRepository{findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}},
		isLead(),
		&mockStandingsCache{},
	)

	err := uc.Execute(context.Background(), DeleteContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	})
	if err == nil {
		t.Fatal("expected 404 error, got nil")
	}
}

func TestDeleteContest_Lead_Succeeds(t *testing.T) {
	uc := newDeleteContestUseCase(scheduledContestFixture(), isLead())

	err := uc.Execute(context.Background(), DeleteContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContest_Admin_Succeeds(t *testing.T) {
	uc := newDeleteContestUseCase(activeContest(), notLead())

	err := uc.Execute(context.Background(), DeleteContestInput{
		CurrentUser: asAdmin(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContest_AnyStatus_Succeeds(t *testing.T) {
	for _, c := range []*domainContest.Contest{scheduledContestFixture(), activeContest(), finishedContest()} {
		uc := newDeleteContestUseCase(c, isLead())
		err := uc.Execute(context.Background(), DeleteContestInput{
			CurrentUser: asCoach(callerID),
			GroupID:     testGroupID,
			ContestID:   testContestID,
		})
		if err != nil {
			t.Errorf("expected nil error for status %s, got: %v", c.Status(time.Now()), err)
		}
	}
}

func TestDeleteContest_InvalidatesStandingsCache(t *testing.T) {
	invalidated := false
	cache := &mockStandingsCache{
		invalidateFn: func(_ context.Context, _ string) error {
			invalidated = true
			return nil
		},
	}
	uc := NewDeleteContestUseCase(
		repoWith(activeContest()),
		isLead(),
		cache,
	)

	_ = uc.Execute(context.Background(), DeleteContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	})
	if !invalidated {
		t.Error("expected standings cache to be invalidated after delete")
	}
}
