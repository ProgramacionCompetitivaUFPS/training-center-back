package team

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func removeInput(requester, target string) RemoveTeamMemberInput {
	return RemoveTeamMemberInput{CurrentUser: asContestant(requester), TeamID: testTeamID, TargetUserID: target}
}

func assertAppError(t *testing.T, err error, kind apperror.Kind, code string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != kind || appErr.Code != code {
		t.Fatalf("expected %s (%v), got %v", code, kind, err)
	}
}

func newRemoveUseCase(teamRepo *mockTeamRepository, memberRepo *mockMemberRepository, checker *mockContestChecker, cleaner *mockScheduledParticipationCleaner) *RemoveTeamMemberUseCase {
	return NewRemoveTeamMemberUseCase(teamRepo, memberRepo, checker, cleaner, &mockTxManager{})
}

func TestRemoveTeamMember_AnyMemberCanRemoveAnother(t *testing.T) {
	// Arrange: memberB did not create the team
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	var deleted string
	memberRepo.deleteByTeamAndUserFn = func(_ string, userID domainShared.UserID) error {
		deleted = userID.Value()
		return nil
	}
	cleaner := &mockScheduledParticipationCleaner{}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, cleaner)

	// Act
	err := uc.Execute(ctx(), removeInput(memberB, memberA))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != memberA {
		t.Errorf("expected %s to be removed, got %q", memberA, deleted)
	}
	if cleaner.removedUserID != memberA {
		t.Errorf("expected the scheduled registrations to lose %s, got %q", memberA, cleaner.removedUserID)
	}
}

func TestRemoveTeamMember_CannotRemoveYourself(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	err := uc.Execute(ctx(), removeInput(memberA, memberA))

	assertAppError(t, err, apperror.KindBadRequest, domainTeam.ErrCodeCannotRemoveSelf)
}

func TestRemoveTeamMember_CannotRemoveYourselfWrittenInCapitals(t *testing.T) {
	// Arrange: the same UUID in upper case is the same person; Postgres would resolve it to the caller
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("the caller must not be deleted by writing their own id in capitals")
		return nil
	}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, strings.ToUpper(memberA)))

	// Assert
	assertAppError(t, err, apperror.KindBadRequest, domainTeam.ErrCodeCannotRemoveSelf)
}

func TestRemoveTeamMember_TargetWrittenInCapitalsIsRemovedByItsCanonicalID(t *testing.T) {
	// Arrange
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	var deleted string
	memberRepo.deleteByTeamAndUserFn = func(_ string, userID domainShared.UserID) error {
		deleted = userID.Value()
		return nil
	}
	cleaner := &mockScheduledParticipationCleaner{}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, cleaner)

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, strings.ToUpper(memberB)))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != memberB || cleaner.removedUserID != memberB {
		t.Errorf("expected the canonical %s everywhere, got deleted=%q cleaned=%q", memberB, deleted, cleaner.removedUserID)
	}
}

func TestRemoveTeamMember_MalformedTargetReturnsNotFound(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	err := uc.Execute(ctx(), removeInput(memberA, "not-a-uuid"))

	assertAppError(t, err, apperror.KindNotFound, domainTeam.ErrCodeNotTeamMember)
}

func TestRemoveTeamMember_NonMemberRequesterIsForbidden(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	err := uc.Execute(ctx(), removeInput(strangerID, memberB))

	assertAppError(t, err, apperror.KindForbidden, domainTeam.ErrCodeNotTeamMember)
}

func TestRemoveTeamMember_TargetNotOnTheTeamReturnsNotFound(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	err := uc.Execute(ctx(), removeInput(memberA, strangerID))

	assertAppError(t, err, apperror.KindNotFound, domainTeam.ErrCodeNotTeamMember)
}

func TestRemoveTeamMember_RefusedWhileTargetIsInAnActiveContest(t *testing.T) {
	// Arrange
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("nobody may be removed while selected in a running contest")
		return nil
	}
	checker := &mockContestChecker{inActiveFn: func(userID, _ string) (bool, error) { return userID == memberB, nil }}
	cleaner := &mockScheduledParticipationCleaner{}
	uc := newRemoveUseCase(teamRepo, memberRepo, checker, cleaner)

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	// Assert: the check is about the removed member, not the requester
	assertAppError(t, err, apperror.KindConflict, domainTeam.ErrCodeCannotRemoveDuringActiveContest)
	if cleaner.called {
		t.Error("the registrations must not be touched when the removal is refused")
	}
}

func TestRemoveTeamMember_CleanupFailureKeepsTheMember(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("the member must not be deleted when the registrations could not be updated")
		return nil
	}
	cleaner := &mockScheduledParticipationCleaner{err: apperror.NewInternal()}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, cleaner)

	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestRemoveTeamMember_LocksTheTeamBeforeCheckingMembership(t *testing.T) {
	// Arrange: two members removing each other at once are only safe if the second one waits for the
	// lock and reads membership afterwards, so no membership lookup may run before the lock is held.
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	locked := false
	teamRepo.lockFn = func(id string) error {
		if id != testTeamID {
			t.Errorf("expected the lock on %s, got %s", testTeamID, id)
		}
		locked = true
		return nil
	}
	lookup := memberRepo.findByTeamAndUserFn
	memberRepo.findByTeamAndUserFn = func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
		if !locked {
			t.Error("membership was read before the team lock was taken")
		}
		return lookup(teamID, userID)
	}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Error("the team lock was never taken")
	}
}

func TestRemoveTeamMember_LockFailureStopsTheRemoval(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	teamRepo.lockFn = func(_ string) error {
		return apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
	}
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("nothing may be deleted when the team could not be locked")
		return nil
	}
	uc := newRemoveUseCase(teamRepo, memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{})

	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	assertAppError(t, err, apperror.KindNotFound, domainTeam.ErrCodeTeamNotFound)
}

type markedTx struct{}

type markingTxManager struct{}

func (markingTxManager) WithTx(c context.Context, fn func(context.Context) error) error {
	return fn(context.WithValue(c, markedTx{}, true))
}

type txAwareChecker struct{ sawTransaction *bool }

func (c txAwareChecker) IsUserInActiveContestForTeam(ctx context.Context, _, _ string) (bool, error) {
	*c.sawTransaction = ctx.Value(markedTx{}) == true
	return false, nil
}

func TestRemoveTeamMember_ChecksTheActiveContestInsideTheTransaction(t *testing.T) {
	// Arrange: the check and the cleanup must read the same NOW(), i.e. run in the same transaction;
	// otherwise a contest that starts between them is skipped by the cleanup after the check let it pass.
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	saw := false
	uc := NewRemoveTeamMemberUseCase(teamRepo, memberRepo, txAwareChecker{sawTransaction: &saw}, &mockScheduledParticipationCleaner{}, markingTxManager{})

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !saw {
		t.Error("the active-contest check ran outside the transaction")
	}
}
