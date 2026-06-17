package team

import (
	"testing"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestRejectInvitation_NotFoundReturns404(t *testing.T) {
	uc := NewRejectInvitationUseCase(&mockInvitationRepository{})
	err := uc.Execute(ctx(), RejectInvitationInput{
		CurrentUser:  asContestant("u1"),
		InvitationID: "missing",
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}

func TestRejectInvitation_WrongUserReturns403(t *testing.T) {
	invRepo := &mockInvitationRepository{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u2"),
				domainShared.RestoreUserID("u1"),
				time.Now()), nil
		},
	}
	uc := NewRejectInvitationUseCase(invRepo)
	err := uc.Execute(ctx(), RejectInvitationInput{
		CurrentUser:  asContestant("u3"),
		InvitationID: "inv1",
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected KindForbidden, got %v", err)
	}
}

func TestRejectInvitation_SuccessDeletes(t *testing.T) {
	invRepo := &mockInvitationRepository{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u2"),
				domainShared.RestoreUserID("u1"),
				time.Now()), nil
		},
	}
	uc := NewRejectInvitationUseCase(invRepo)
	if err := uc.Execute(ctx(), RejectInvitationInput{
		CurrentUser:  asContestant("u2"),
		InvitationID: "inv1",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
