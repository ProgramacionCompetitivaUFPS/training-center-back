package team

import (
	"testing"
	"time"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/internal/testutil"
)

func makeTeam(id, name, createdBy string, now time.Time) *domainTeam.Team {
	return domainTeam.RestoreTeam(id, domainTeam.RestoreTeamName(name), shared.RestoreUserID(createdBy), now)
}

func makeMember(id, teamID, userID string, now time.Time) *domainTeam.TeamMember {
	return domainTeam.RestoreTeamMember(id, teamID, shared.RestoreUserID(userID), now)
}

func TestGetTeam_AnyAuthenticatedUserCanView(t *testing.T) {
	now := time.Now()
	team := makeTeam("t1", "Alpha", "creator-1", now)
	member := makeMember("m1", "t1", "creator-1", now)

	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) { return team, nil },
	}
	memberRepo := &mockMemberRepository{
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{member}, nil
		},
	}
	userProv := &mockUserProvider{
		displaysFn: func(ids []string) (map[string]*UserDisplay, error) {
			m := make(map[string]*UserDisplay)
			for _, id := range ids {
				m[id] = &UserDisplay{Nickname: "nick_" + id}
			}
			return m, nil
		},
	}

	uc := NewGetTeamUseCase(teamRepo, memberRepo, &mockInvitationRepository{}, userProv)
	out, err := uc.Execute(ctx(), GetTeamInput{CurrentUser: testutil.AsContestant("stranger-99"), TeamID: "t1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "t1" {
		t.Errorf("ID = %q, want t1", out.ID)
	}
	if len(out.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(out.Members))
	}
	if out.Members[0].Nickname != "nick_creator-1" {
		t.Errorf("Nickname = %q, want nick_creator-1", out.Members[0].Nickname)
	}
}

func TestGetTeam_MemberSeesInvitations(t *testing.T) {
	now := time.Now()
	team := makeTeam("t1", "Alpha", "creator-1", now)
	inv := domainTeam.RestoreTeamInvitation("inv-1", "t1", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("creator-1"), now)

	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) { return team, nil },
	}
	memberRepo := &mockMemberRepository{
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{}, nil
		},
	}
	invRepo := &mockInvitationRepository{
		findByTeamFn: func(teamID string) ([]*domainTeam.TeamInvitation, error) {
			return []*domainTeam.TeamInvitation{inv}, nil
		},
	}

	uc := NewGetTeamUseCase(teamRepo, memberRepo, invRepo, &mockUserProvider{})
	// creator-1 is the team creator → counts as member
	out, err := uc.Execute(ctx(), GetTeamInput{CurrentUser: testutil.AsContestant("creator-1"), TeamID: "t1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.PendingInvitations) != 1 {
		t.Fatalf("expected 1 pending invitation, got %d", len(out.PendingInvitations))
	}
	if out.PendingInvitations[0].ID != "inv-1" {
		t.Errorf("invitation ID = %q, want inv-1", out.PendingInvitations[0].ID)
	}
}

func TestGetTeam_NonMemberSeesNoInvitations(t *testing.T) {
	now := time.Now()
	team := makeTeam("t1", "Alpha", "creator-1", now)
	inv := domainTeam.RestoreTeamInvitation("inv-1", "t1", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("creator-1"), now)

	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) { return team, nil },
	}
	memberRepo := &mockMemberRepository{
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{}, nil
		},
	}
	invRepo := &mockInvitationRepository{
		findByTeamFn: func(teamID string) ([]*domainTeam.TeamInvitation, error) {
			return []*domainTeam.TeamInvitation{inv}, nil
		},
	}

	uc := NewGetTeamUseCase(teamRepo, memberRepo, invRepo, &mockUserProvider{})
	out, err := uc.Execute(ctx(), GetTeamInput{CurrentUser: testutil.AsContestant("stranger-99"), TeamID: "t1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.PendingInvitations) != 0 {
		t.Errorf("expected 0 pending invitations for non-member, got %d", len(out.PendingInvitations))
	}
}

func TestGetTeam_TeamNotFoundReturns404(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
		},
	}

	uc := NewGetTeamUseCase(teamRepo, &mockMemberRepository{}, &mockInvitationRepository{}, &mockUserProvider{})
	_, err := uc.Execute(ctx(), GetTeamInput{CurrentUser: testutil.AsContestant("u1"), TeamID: "missing"})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}
