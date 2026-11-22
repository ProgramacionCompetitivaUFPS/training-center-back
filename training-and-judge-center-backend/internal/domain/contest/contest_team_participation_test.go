package contest_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
)

func TestNewContestTeamParticipation_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	p, err := contest.NewContestTeamParticipation("part-1", "contest-1", "team-1", []string{"user-a", "user-b"}, now)

	if err != nil {
		t.Fatalf("NewContestTeamParticipation: %v", err)
	}
	if p.ID() != "part-1" || p.ContestID() != "contest-1" || p.TeamID() != "team-1" {
		t.Errorf("fields = %s/%s/%s, want part-1/contest-1/team-1", p.ID(), p.ContestID(), p.TeamID())
	}
	if want := []string{"user-a", "user-b"}; !reflect.DeepEqual(p.SelectedMembers(), want) {
		t.Errorf("SelectedMembers = %v, want %v", p.SelectedMembers(), want)
	}
	if want := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC); !p.RegisteredAt().Equal(want) || p.RegisteredAt().Location() != time.UTC {
		t.Errorf("RegisteredAt = %v, want %v in UTC", p.RegisteredAt(), want)
	}
}

func TestNewContestTeamParticipation_MissingIdentifiers_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name                string
		id, contestID, team string
	}{
		{"an empty id", "", "contest-1", "team-1"},
		{"an empty contest id", "part-1", "", "team-1"},
		{"an empty team id", "part-1", "contest-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := contest.NewContestTeamParticipation(tt.id, tt.contestID, tt.team, nil, fixedNow)

			assertInternalError(t, tt.name, err)
			if p != nil {
				t.Errorf("expected no participation, got %+v", p)
			}
		})
	}
}

func TestContestTeamParticipation_SelectedMembers_AreIsolatedFromTheCaller(t *testing.T) {
	input := []string{"user-a", "user-b"}
	p, err := contest.NewContestTeamParticipation("part-1", "contest-1", "team-1", input, fixedNow)
	if err != nil {
		t.Fatalf("NewContestTeamParticipation: %v", err)
	}

	input[0] = "tampered-input"
	p.SelectedMembers()[1] = "tampered-output"

	if want := []string{"user-a", "user-b"}; !reflect.DeepEqual(p.SelectedMembers(), want) {
		t.Errorf("SelectedMembers = %v, want %v: the caller's slices must not alias the stored one", p.SelectedMembers(), want)
	}
}

func TestContestTeamParticipation_UpdateSelectedMembers_ReplacesAndCopies(t *testing.T) {
	p, err := contest.NewContestTeamParticipation("part-1", "contest-1", "team-1", []string{"user-a"}, fixedNow)
	if err != nil {
		t.Fatalf("NewContestTeamParticipation: %v", err)
	}
	replacement := []string{"user-c", "user-d"}

	p.UpdateSelectedMembers(replacement)
	replacement[0] = "tampered"

	if want := []string{"user-c", "user-d"}; !reflect.DeepEqual(p.SelectedMembers(), want) {
		t.Errorf("SelectedMembers = %v, want %v", p.SelectedMembers(), want)
	}
}

func TestRestoreContestTeamParticipation_CopiesTheMembersAndNormalisesTheInstant(t *testing.T) {
	members := []string{"user-a"}
	registeredAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	p := contest.RestoreContestTeamParticipation("part-1", "contest-1", "team-1", members, registeredAt)
	members[0] = "tampered"

	if want := []string{"user-a"}; !reflect.DeepEqual(p.SelectedMembers(), want) {
		t.Errorf("SelectedMembers = %v, want %v", p.SelectedMembers(), want)
	}
	if want := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC); !p.RegisteredAt().Equal(want) || p.RegisteredAt().Location() != time.UTC {
		t.Errorf("RegisteredAt = %v, want %v in UTC", p.RegisteredAt(), want)
	}
}
