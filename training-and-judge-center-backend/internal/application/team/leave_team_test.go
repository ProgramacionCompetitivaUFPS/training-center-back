package team

import (
	"testing"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestLeaveTeam_NonMemberReturns404(t *testing.T) {
	uc := NewLeaveTeamUseCase(&mockMemberRepository{}, &mockContestChecker{})
	err := uc.Execute(ctx(), LeaveTeamInput{
		CurrentUser: asContestant("u1"),
		TeamID:      "t1",
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}

func TestLeaveTeam_ActiveContestReturns409(t *testing.T) {
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	checker := &mockContestChecker{
		inActiveFn: func(_, _ string) (bool, error) { return true, nil },
	}
	uc := NewLeaveTeamUseCase(memberRepo, checker)
	err := uc.Execute(ctx(), LeaveTeamInput{
		CurrentUser: asContestant("u1"),
		TeamID:      "t1",
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindConflict {
		t.Errorf("expected KindConflict, got %v", err)
	}
	if ae.Code != domainTeam.ErrCodeCannotLeaveDuringActiveContest {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeCannotLeaveDuringActiveContest, ae.Code)
	}
}

func TestLeaveTeam_SuccessDeletes(t *testing.T) {
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	uc := NewLeaveTeamUseCase(memberRepo, &mockContestChecker{})
	if err := uc.Execute(ctx(), LeaveTeamInput{
		CurrentUser: asContestant("u1"),
		TeamID:      "t1",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
