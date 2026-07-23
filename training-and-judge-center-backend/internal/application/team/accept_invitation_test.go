package team

import (
	"testing"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestAcceptInvitation_NotFoundReturns404(t *testing.T) {
	uc := NewAcceptInvitationUseCase(&mockInvitationRepository{}, &mockMemberRepository{}, &mockTxManager{})
	_, err := uc.Execute(ctx(), AcceptInvitationInput{
		CurrentUser:  asContestant("u1"),
		InvitationID: "missing",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}

func TestAcceptInvitation_WrongUserReturns403(t *testing.T) {
	invRepo := &mockInvitationRepository{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u2"),
				domainShared.RestoreUserID("u1"),
				time.Now()), nil
		},
	}
	uc := NewAcceptInvitationUseCase(invRepo, &mockMemberRepository{}, &mockTxManager{})
	_, err := uc.Execute(ctx(), AcceptInvitationInput{
		CurrentUser:  asContestant("u3"),
		InvitationID: "inv1",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected KindForbidden, got %v", err)
	}
}

func TestAcceptInvitation_SuccessCreatesMember(t *testing.T) {
	var deletedID string
	invRepo := &mockInvitationRepository{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u2"),
				domainShared.RestoreUserID("u1"),
				time.Now()), nil
		},
		deleteFn: func(id string) error {
			deletedID = id
			return nil
		},
	}
	uc := NewAcceptInvitationUseCase(invRepo, &mockMemberRepository{}, &mockTxManager{})
	out, err := uc.Execute(ctx(), AcceptInvitationInput{
		CurrentUser:  asContestant("u2"),
		InvitationID: "inv1",
	}, time.Now())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.JoinedAt.IsZero() {
		t.Error("expected non-zero JoinedAt")
	}
	if deletedID != "inv1" {
		t.Errorf("expected Delete called with inv1, got %q", deletedID)
	}
}
