package team

import (
	"testing"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

func TestListMyInvitations_EmptyList(t *testing.T) {
	uc := NewListMyInvitationsUseCase(&mockInvitationRepository{}, &mockTeamRepository{}, &mockUserProvider{})
	out, err := uc.Execute(ctx(), ListMyInvitationsInput{CurrentUser: asContestant("u1")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Invitations) != 0 {
		t.Errorf("expected empty list, got %d", len(out.Invitations))
	}
}

func TestListMyInvitations_ReturnsPendingInvitations(t *testing.T) {
	now := time.Now()
	inv := domainTeam.RestoreTeamInvitation("inv1", "t1",
		domainShared.RestoreUserID("u2"),
		domainShared.RestoreUserID("u1"),
		now)

	invRepo := &mockInvitationRepository{
		findByInviteeFn: func(_ domainShared.UserID) ([]*domainTeam.TeamInvitation, error) {
			return []*domainTeam.TeamInvitation{inv}, nil
		},
	}
	teamRepo := &mockTeamRepository{
		findByIDsFn: func(_ []string) ([]*domainTeam.Team, error) {
			return []*domainTeam.Team{makeTeam("t1", "Alpha", "u1", now)}, nil
		},
	}

	uc := NewListMyInvitationsUseCase(invRepo, teamRepo, &mockUserProvider{})
	out, err := uc.Execute(ctx(), ListMyInvitationsInput{CurrentUser: asContestant("u2")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(out.Invitations))
	}
	if out.Invitations[0].Team.ID != "t1" {
		t.Errorf("Team.ID = %q, want t1", out.Invitations[0].Team.ID)
	}
	if out.Invitations[0].Team.Name != "Alpha" {
		t.Errorf("Team.Name = %q, want Alpha", out.Invitations[0].Team.Name)
	}
}
