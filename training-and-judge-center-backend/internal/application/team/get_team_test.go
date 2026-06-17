package team

import (
	"testing"
	"time"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func makeTeam(id, name, createdBy string, now time.Time) *domainTeam.Team {
	return domainTeam.RestoreTeam(id, domainTeam.RestoreTeamName(name), shared.RestoreUserID(createdBy), now)
}

func makeMember(id, teamID, userID string, now time.Time) *domainTeam.TeamMember {
	return domainTeam.RestoreTeamMember(id, teamID, shared.RestoreUserID(userID), now)
}

func TestGetTeam_MemberCanView(t *testing.T) {
	now := time.Now()
	team := makeTeam("t1", "Alpha", "creator-1", now)
	member := makeMember("m1", "t1", "user-1", now)

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

	uc := NewGetTeamUseCase(teamRepo, memberRepo, userProv)
	out, err := uc.Execute(ctx(), GetTeamInput{TeamID: "t1", CurrentUser: asContestant("user-1")})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "t1" {
		t.Errorf("ID = %q, want t1", out.ID)
	}
	if len(out.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(out.Members))
	}
	if out.Members[0].Nickname != "nick_user-1" {
		t.Errorf("Nickname = %q, want nick_user-1", out.Members[0].Nickname)
	}
}

func TestGetTeam_NonMemberGets403(t *testing.T) {
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

	uc := NewGetTeamUseCase(teamRepo, memberRepo, &mockUserProvider{})
	_, err := uc.Execute(ctx(), GetTeamInput{TeamID: "t1", CurrentUser: asContestant("outsider")})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected KindForbidden, got %v", err)
	}
	if ae.Code != domainTeam.ErrCodeNotTeamMember {
		t.Errorf("expected NOT_TEAM_MEMBER, got %s", ae.Code)
	}
}

func TestGetTeam_AdminCanViewWithoutMembership(t *testing.T) {
	now := time.Now()
	team := makeTeam("t1", "Alpha", "creator-1", now)

	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) { return team, nil },
	}
	memberRepo := &mockMemberRepository{
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{}, nil
		},
	}

	uc := NewGetTeamUseCase(teamRepo, memberRepo, &mockUserProvider{})
	out, err := uc.Execute(ctx(), GetTeamInput{TeamID: "t1", CurrentUser: asAdmin("admin-1")})

	if err != nil {
		t.Fatalf("admin should be able to view any team, got: %v", err)
	}
	if out.ID != "t1" {
		t.Errorf("ID = %q, want t1", out.ID)
	}
}

func TestGetTeam_TeamNotFoundReturns404(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
		},
	}
	memberRepo := &mockMemberRepository{}

	uc := NewGetTeamUseCase(teamRepo, memberRepo, &mockUserProvider{})
	_, err := uc.Execute(ctx(), GetTeamInput{TeamID: "missing", CurrentUser: asContestant("user-1")})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}
