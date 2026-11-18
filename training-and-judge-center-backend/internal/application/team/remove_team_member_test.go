package team

import (
	"errors"
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

func TestRemoveTeamMember_AnyMemberCanRemoveAnother(t *testing.T) {
	// Arrange: memberB did not create the team
	_, memberRepo := teamWithMembers(memberA, memberB)
	var deleted string
	memberRepo.deleteByTeamAndUserFn = func(_ string, userID domainShared.UserID) error {
		deleted = userID.Value()
		return nil
	}
	cleaner := &mockScheduledParticipationCleaner{}
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, cleaner, &mockTxManager{})

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
	_, memberRepo := teamWithMembers(memberA, memberB)
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{}, &mockTxManager{})

	err := uc.Execute(ctx(), removeInput(memberA, memberA))

	assertAppError(t, err, apperror.KindBadRequest, domainTeam.ErrCodeCannotRemoveSelf)
}

func TestRemoveTeamMember_NonMemberRequesterIsForbidden(t *testing.T) {
	_, memberRepo := teamWithMembers(memberA, memberB)
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{}, &mockTxManager{})

	err := uc.Execute(ctx(), removeInput(strangerID, memberB))

	assertAppError(t, err, apperror.KindForbidden, domainTeam.ErrCodeNotTeamMember)
}

func TestRemoveTeamMember_TargetNotOnTheTeamReturnsNotFound(t *testing.T) {
	_, memberRepo := teamWithMembers(memberA, memberB)
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{}, &mockTxManager{})

	err := uc.Execute(ctx(), removeInput(memberA, strangerID))

	assertAppError(t, err, apperror.KindNotFound, domainTeam.ErrCodeNotTeamMember)
}

func TestRemoveTeamMember_RefusedWhileTargetIsInAnActiveContest(t *testing.T) {
	// Arrange
	_, memberRepo := teamWithMembers(memberA, memberB)
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("nobody may be removed while selected in a running contest")
		return nil
	}
	checker := &mockContestChecker{inActiveFn: func(userID, _ string) (bool, error) { return userID == memberB, nil }}
	cleaner := &mockScheduledParticipationCleaner{}
	uc := NewRemoveTeamMemberUseCase(memberRepo, checker, cleaner, &mockTxManager{})

	// Act
	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	// Assert: the check is about the removed member, not the requester
	assertAppError(t, err, apperror.KindConflict, domainTeam.ErrCodeCannotRemoveDuringActiveContest)
	if cleaner.called {
		t.Error("the registrations must not be touched when the removal is refused")
	}
}

func TestRemoveTeamMember_CleanupFailureKeepsTheMember(t *testing.T) {
	_, memberRepo := teamWithMembers(memberA, memberB)
	memberRepo.deleteByTeamAndUserFn = func(_ string, _ domainShared.UserID) error {
		t.Error("the member must not be deleted when the registrations could not be updated")
		return nil
	}
	cleaner := &mockScheduledParticipationCleaner{err: apperror.NewInternal()}
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, cleaner, &mockTxManager{})

	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestRemoveTeamMember_RequesterRemovedMeanwhileIsForbidden(t *testing.T) {
	// Arrange: two members removing each other at once must not empty the team. The requester is
	// still on it when the request starts, and gone by the time the transaction runs.
	calls := 0
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == memberA {
				calls++
				if calls > 1 {
					return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
				}
			}
			return makeMember("m-"+userID.Value(), teamID, userID.Value(), testNow), nil
		},
		deleteByTeamAndUserFn: func(_ string, _ domainShared.UserID) error {
			t.Error("the removal must not go through")
			return nil
		},
	}
	uc := NewRemoveTeamMemberUseCase(memberRepo, &mockContestChecker{}, &mockScheduledParticipationCleaner{}, &mockTxManager{})

	err := uc.Execute(ctx(), removeInput(memberA, memberB))

	assertAppError(t, err, apperror.KindForbidden, domainTeam.ErrCodeNotTeamMember)
}
