package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListRegistrationsUseCase(
	repo *mockContestRepository,
	reg *mockRegistrationRepository,
	member *mockGroupMemberProvider,
	nicknames *mockNicknameProvider,
) *ListContestRegistrationsUseCase {
	return NewListContestRegistrationsUseCase(repo, reg, mockTeamParticipRepo(), groupProviderWith(visibleGroup()), member, nicknames, &mockTeamDisplayProvider{})
}

func validListRegistrationsInput() ListContestRegistrationsInput {
	return ListContestRegistrationsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	}
}

func TestListContestRegistrations_HappyPath(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()),
			}, 1, nil
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("expected total=1, got %d", out.Total)
	}
	if len(out.Registrations) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(out.Registrations))
	}
	if out.Registrations[0].Nickname == "" {
		t.Error("expected non-empty nickname")
	}
}

func TestListContestRegistrations_EmptyList(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 0 || len(out.Registrations) != 0 {
		t.Errorf("expected empty list, got total=%d len=%d", out.Total, len(out.Registrations))
	}
}

func TestListContestRegistrations_ContestNotFound(t *testing.T) {
	uc := newListRegistrationsUseCase(&mockContestRepository{}, mockRegistrations(), isMemberNotLead(), mockNicknames())

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error for missing contest")
	}
}

func TestListContestRegistrations_VisibleGroupNonMemberCanView(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead(), mockNicknames())

	_, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("non-member of a visible group should see registrations, got %v", err)
	}
}

func TestListContestRegistrations_HiddenGroupNonMember_Returns404(t *testing.T) {
	uc := NewListContestRegistrationsUseCase(
		repoWith(newTestContest(otherID)), mockRegistrations(), mockTeamParticipRepo(),
		groupProviderWith(hiddenGroup()), notLead(), mockNicknames(), &mockTeamDisplayProvider{},
	)

	_, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if !isNotFound(err) {
		t.Fatalf("expected not found for non-member of a hidden group, got %v", err)
	}
}

func TestListContestRegistrations_LeadCanView(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("lead should be able to view registrations, got %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestListContestRegistrations_NicknameProviderError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()),
			}, 1, nil
		},
	}
	nicknames := &mockNicknameProvider{
		getFn: func(_ context.Context, _ []string) (map[string]string, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), nicknames)

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error from failed GetNicknamesByIDs")
	}
}

func TestListContestRegistrations_ListError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return nil, 0, apperror.NewInternal()
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), mockNicknames())

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error from failed ListByContest")
	}
}

func TestListContestRegistrations_TeamCountsAsOneEntryWithMembers(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()),
			}, 1, nil
		},
	}
	teamRepo := &mockTeamParticipationRepository{
		listFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestTeamParticipation, int, error) {
			return []*domainContest.ContestTeamParticipation{
				domainContest.RestoreContestTeamParticipation("tp1", testContestID, "team-1", []string{otherID, "member-2"}, time.Now()),
			}, 1, nil
		},
	}
	teamDisplays := &mockTeamDisplayProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*TeamDisplay, error) {
			return map[string]*TeamDisplay{"team-1": {ID: "team-1", Name: "Los Ñoños"}}, nil
		},
	}
	uc := NewListContestRegistrationsUseCase(
		repoWith(newTestContest(otherID)), reg, teamRepo, groupProviderWith(visibleGroup()), isMemberNotLead(), mockNicknames(), teamDisplays,
	)

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 2 {
		t.Fatalf("expected total=2 (1 individual + 1 team, team counted once), got %d", out.Total)
	}

	var teamEntry *RegistrationEntry
	for i := range out.Registrations {
		if out.Registrations[i].Type == "TEAM" {
			teamEntry = &out.Registrations[i]
		}
	}
	if teamEntry == nil {
		t.Fatal("expected one TEAM entry")
	}
	if teamEntry.TeamName != "Los Ñoños" {
		t.Errorf("teamName=%q, want %q", teamEntry.TeamName, "Los Ñoños")
	}
	if len(teamEntry.Members) != 2 {
		t.Errorf("expected 2 members in the team entry, got %d", len(teamEntry.Members))
	}
}

func TestListContestRegistrations_SearchMatchesNicknameTeamNameAndMembers(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, "solo-1", time.Now()),
				domainContest.RestoreContestRegistration("r2", testContestID, "solo-2", time.Now()),
			}, 2, nil
		},
	}
	teamRepo := &mockTeamParticipationRepository{
		listFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestTeamParticipation, int, error) {
			return []*domainContest.ContestTeamParticipation{
				domainContest.RestoreContestTeamParticipation("tp1", testContestID, "team-1", []string{"member-a", "member-b"}, time.Now()),
			}, 1, nil
		},
	}
	teamDisplays := &mockTeamDisplayProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*TeamDisplay, error) {
			return map[string]*TeamDisplay{"team-1": {ID: "team-1", Name: "Los Ñoños"}}, nil
		},
	}
	uc := NewListContestRegistrationsUseCase(
		repoWith(newTestContest(otherID)), reg, teamRepo, groupProviderWith(visibleGroup()), isMemberNotLead(), mockNicknames(), teamDisplays,
	)

	cases := []struct {
		search string
		want   int
	}{
		{"", 3},
		{"SOLO-1", 1},        // individual nickname, case-insensitive
		{"ñoños", 1},         // team name
		{"nick_member-b", 1}, // team member nickname
		{"solo", 2},
		{"nadie", 0},
	}
	for _, tc := range cases {
		in := validListRegistrationsInput()
		in.Search = tc.search
		out, err := uc.Execute(context.Background(), in)
		if err != nil {
			t.Fatalf("search=%q: unexpected error: %v", tc.search, err)
		}
		if out.Total != tc.want || len(out.Registrations) != tc.want {
			t.Errorf("search=%q: total=%d len=%d, want %d", tc.search, out.Total, len(out.Registrations), tc.want)
		}
	}
}

func groupProviderWith(g *GroupInfo) *mockGroupProvider {
	return &mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) { return g, nil }}
}

func isNotFound(err error) bool {
	ae, ok := err.(*apperror.AppError)
	return ok && ae.Kind == apperror.KindNotFound
}
