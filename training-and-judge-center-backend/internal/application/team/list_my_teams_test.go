package team

import (
	"testing"
	"time"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func TestListMyTeams_ReturnsTeamsForMember(t *testing.T) {
	now := time.Now()
	uid := shared.RestoreUserID("user-1")
	teamID := "team-1"

	member := domainTeam.RestoreTeamMember("m1", teamID, uid, now.Add(-time.Hour))
	team := domainTeam.RestoreTeam(teamID, domainTeam.RestoreTeamName("Alpha"), uid, now.Add(-2*time.Hour))

	memberRepo := &mockMemberRepository{
		findByUserFn: func(_ shared.UserID) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{member}, nil
		},
		bulkCountFn: func(_ []string) (map[string]int, error) {
			return map[string]int{teamID: 3}, nil
		},
	}
	teamRepo := &mockTeamRepository{
		findByIDsFn: func(_ []string) ([]*domainTeam.Team, error) {
			return []*domainTeam.Team{team}, nil
		},
	}

	uc := NewListMyTeamsUseCase(teamRepo, memberRepo)
	out, err := uc.Execute(ctx(), ListMyTeamsInput{CurrentUser: asContestant("user-1"), Page: 1, Limit: 20})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(out.Teams))
	}
	got := out.Teams[0]
	if got.ID != teamID {
		t.Errorf("ID = %q, want %q", got.ID, teamID)
	}
	if got.Name != "Alpha" {
		t.Errorf("Name = %q, want %q", got.Name, "Alpha")
	}
	if got.MemberCount != 3 {
		t.Errorf("MemberCount = %d, want 3", got.MemberCount)
	}
}

func TestListMyTeams_EmptyWhenNoTeams(t *testing.T) {
	memberRepo := &mockMemberRepository{
		findByUserFn: func(_ shared.UserID) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{}, nil
		},
	}
	teamRepo := &mockTeamRepository{}

	uc := NewListMyTeamsUseCase(teamRepo, memberRepo)
	out, err := uc.Execute(ctx(), ListMyTeamsInput{CurrentUser: asContestant("user-1"), Page: 1, Limit: 20})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Teams) != 0 {
		t.Errorf("expected 0 teams, got %d", len(out.Teams))
	}
	if out.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", out.TotalCount)
	}
}

func TestListMyTeams_InvalidPaginationReturns400(t *testing.T) {
	memberRepo := &mockMemberRepository{}
	teamRepo := &mockTeamRepository{}

	uc := NewListMyTeamsUseCase(teamRepo, memberRepo)
	_, err := uc.Execute(ctx(), ListMyTeamsInput{CurrentUser: asContestant("user-1"), Page: 0, Limit: 20})

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
