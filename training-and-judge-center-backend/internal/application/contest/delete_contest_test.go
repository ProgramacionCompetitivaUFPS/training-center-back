package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
		0, false, false, false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func lockedContestFixture(c *domainContest.Contest) *domainContest.Contest {
	c.SetLocked(true, testNow)
	return c
}

func deleteInputFor(user appshared.CurrentUser) DeleteContestInput {
	return DeleteContestInput{
		CurrentUser: user,
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Now:         time.Now(),
	}
}

func assertDeleteRejected(t *testing.T, err error, wantCode string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != wantCode {
		t.Fatalf("expected %s, got %v", wantCode, err)
	}
}

func TestDeleteContest_NotLead_Returns403(t *testing.T) {
	uc := newDeleteContestUseCase(scheduledContestFixture(), isMemberNotLead())

	err := uc.Execute(context.Background(), deleteInputFor(asContestant(callerID)))

	assertDeleteRejected(t, err, ErrCodeInsufficientPermissions)
}

func TestDeleteContest_ContestNotFound_Returns404(t *testing.T) {
	uc := NewDeleteContestUseCase(
		&mockContestRepository{findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}},
		isLead(),
		&mockStandingsCache{},
	)

	err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

	assertDeleteRejected(t, err, domainContest.ErrCodeContestNotFound)
}

func TestDeleteContest_Lead_Succeeds(t *testing.T) {
	uc := newDeleteContestUseCase(scheduledContestFixture(), isLead())

	err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContest_Admin_Succeeds(t *testing.T) {
	uc := newDeleteContestUseCase(scheduledContestFixture(), notLead())

	err := uc.Execute(context.Background(), deleteInputFor(asAdmin(callerID)))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContest_ScheduledOrFinished_Succeeds(t *testing.T) {
	for _, c := range []*domainContest.Contest{scheduledContestFixture(), finishedContest()} {
		uc := newDeleteContestUseCase(c, isLead())

		err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

		if err != nil {
			t.Errorf("expected nil error for status %s, got: %v", c.Status(time.Now()), err)
		}
	}
}

func TestDeleteContest_ActiveContest_Returns409(t *testing.T) {
	cases := map[string]struct {
		user     appshared.CurrentUser
		provider *mockGroupMemberProvider
	}{
		"lead":  {asCoach(callerID), isLead()},
		"admin": {asAdmin(callerID), notLead()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deleted := false
			repo := repoWith(activeContest())
			repo.deleteFn = func(_ context.Context, _ string) error {
				deleted = true
				return nil
			}
			uc := NewDeleteContestUseCase(repo, tc.provider, &mockStandingsCache{})

			err := uc.Execute(context.Background(), deleteInputFor(tc.user))

			assertDeleteRejected(t, err, domainContest.ErrCodeContestIsActive)
			if deleted {
				t.Error("an active contest must not reach the repository's Delete")
			}
		})
	}
}

func TestDeleteContest_LockedContest_Returns403ForEveryone(t *testing.T) {
	cases := map[string]struct {
		user     appshared.CurrentUser
		provider *mockGroupMemberProvider
	}{
		"owner who is lead":     {asCoach(callerID), isLead()},
		"lead who is not owner": {asCoach(otherID), isLead()},
		"admin":                 {asAdmin(otherID), notLead()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deleted, invalidated := false, false
			repo := repoWith(lockedContestFixture(scheduledContestFixture()))
			repo.deleteFn = func(_ context.Context, _ string) error {
				deleted = true
				return nil
			}
			cache := &mockStandingsCache{invalidateFn: func(_ context.Context, _ string) error {
				invalidated = true
				return nil
			}}
			uc := NewDeleteContestUseCase(repo, tc.provider, cache)

			err := uc.Execute(context.Background(), deleteInputFor(tc.user))

			assertDeleteRejected(t, err, domainContest.ErrCodeContestLocked)
			if deleted || invalidated {
				t.Errorf("a locked contest must be left untouched (deleted=%v, cacheInvalidated=%v)", deleted, invalidated)
			}
		})
	}
}

func TestDeleteContest_RepositoryRefusesAfterTheCheck_ReturnsItsErrorAndKeepsTheCache(t *testing.T) {
	cases := map[string]*apperror.AppError{
		"locked after it was loaded":    apperror.NewForbidden(domainContest.ErrCodeContestLocked, "contest is locked"),
		"changed while being deleted":   apperror.NewConflict(domainContest.ErrCodeContestConflict, "contest changed"),
		"deleted by someone else first": apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found"),
	}
	for name, refusal := range cases {
		t.Run(name, func(t *testing.T) {
			invalidated := false
			repo := repoWith(scheduledContestFixture())
			repo.deleteFn = func(_ context.Context, _ string) error { return refusal }
			cache := &mockStandingsCache{invalidateFn: func(_ context.Context, _ string) error {
				invalidated = true
				return nil
			}}
			uc := NewDeleteContestUseCase(repo, isLead(), cache)

			err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

			assertDeleteRejected(t, err, refusal.Code)
			if invalidated {
				t.Error("the standings cache must not be invalidated when nothing was deleted")
			}
		})
	}
}

func TestDeleteContest_LockedContest_NonLeadStillGetsInsufficientPermissions(t *testing.T) {
	uc := newDeleteContestUseCase(lockedContestFixture(scheduledContestFixture()), isMemberNotLead())

	err := uc.Execute(context.Background(), deleteInputFor(asContestant(callerID)))

	assertDeleteRejected(t, err, ErrCodeInsufficientPermissions)
}

func TestDeleteContest_LockedAndActive_ReportsTheLockFirst(t *testing.T) {
	uc := newDeleteContestUseCase(lockedContestFixture(activeContest()), isLead())

	err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

	assertDeleteRejected(t, err, domainContest.ErrCodeContestLocked)
}

func TestDeleteContest_UnlockedAgain_Succeeds(t *testing.T) {
	c := lockedContestFixture(scheduledContestFixture())
	c.SetLocked(false, testNow)
	uc := newDeleteContestUseCase(c, isLead())

	err := uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		repoWith(scheduledContestFixture()),
		isLead(),
		cache,
	)

	_ = uc.Execute(context.Background(), deleteInputFor(asCoach(callerID)))

	if !invalidated {
		t.Error("expected standings cache to be invalidated after delete")
	}
}
