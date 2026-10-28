package team

import (
	"errors"
	"testing"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func memberOf() *mockMemberRepository {
	return &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
}

func TestGetLeaveImpact_NotAMember_ReturnsNotFound(t *testing.T) {
	uc := NewGetLeaveImpactUseCase(&mockMemberRepository{}, &mockScheduledParticipationCleaner{})

	_, err := uc.Execute(ctx(), GetLeaveImpactInput{CurrentUser: asContestant("u1"), TeamID: "t1"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainTeam.ErrCodeNotTeamMember {
		t.Fatalf("expected NOT_TEAM_MEMBER, got %v", err)
	}
}

func TestGetLeaveImpact_ListsTheRegistrationsThatWouldBeCancelled(t *testing.T) {
	cleaner := &mockScheduledParticipationCleaner{atRisk: []RegistrationAtRisk{{ContestID: "c1", ContestName: "Taller ICPC", GroupID: "g1"}}}
	uc := NewGetLeaveImpactUseCase(memberOf(), cleaner)

	out, err := uc.Execute(ctx(), GetLeaveImpactInput{CurrentUser: asContestant("u1"), TeamID: "t1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.RegistrationsToCancel) != 1 || out.RegistrationsToCancel[0].ContestName != "Taller ICPC" {
		t.Errorf("expected the registration in Taller ICPC, got %+v", out.RegistrationsToCancel)
	}
	if cleaner.called {
		t.Error("the preview must not change anything")
	}
}

func TestGetLeaveImpact_NothingAtRisk_ReturnsEmpty(t *testing.T) {
	uc := NewGetLeaveImpactUseCase(memberOf(), &mockScheduledParticipationCleaner{})

	out, err := uc.Execute(ctx(), GetLeaveImpactInput{CurrentUser: asContestant("u1"), TeamID: "t1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.RegistrationsToCancel) != 0 {
		t.Errorf("expected nothing at risk, got %+v", out.RegistrationsToCancel)
	}
}
