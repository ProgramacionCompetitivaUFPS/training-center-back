package team_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/team"
)

func TestNewTeam_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	tm, err := team.NewTeam("team-1", mustTeamName(t, "Alpha"), shared.RestoreUserID("user-1"), localNow)

	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	if tm.ID() != "team-1" || tm.Name().Value() != "Alpha" || tm.CreatedBy().Value() != "user-1" {
		t.Errorf("fields = %s/%s/%s, want team-1/Alpha/user-1", tm.ID(), tm.Name().Value(), tm.CreatedBy().Value())
	}
	assertInUTC(t, "CreatedAt", tm.CreatedAt(), utcOfNow)
}

func TestNewTeam_EmptyID_IsAnInternalError(t *testing.T) {
	tm, err := team.NewTeam("", mustTeamName(t, "Alpha"), shared.RestoreUserID("user-1"), fixedNow)

	assertInternalError(t, "empty id", err)
	if tm != nil {
		t.Errorf("expected no team, got %+v", tm)
	}
}

func TestTeam_Rename_ReplacesOnlyTheName(t *testing.T) {
	tm, err := team.NewTeam("team-1", mustTeamName(t, "Alpha"), shared.RestoreUserID("user-1"), fixedNow)
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}

	tm.Rename(mustTeamName(t, "Beta"))

	if tm.Name().Value() != "Beta" {
		t.Errorf("Name = %q, want Beta", tm.Name().Value())
	}
	if tm.ID() != "team-1" || tm.CreatedBy().Value() != "user-1" || !tm.CreatedAt().Equal(fixedNow) {
		t.Errorf("Rename changed more than the name: %s/%s/%v", tm.ID(), tm.CreatedBy().Value(), tm.CreatedAt())
	}
}

func TestRestoreTeam_KeepsTheStoredValues(t *testing.T) {
	tm := team.RestoreTeam("team-1", team.RestoreTeamName("Stored"), shared.RestoreUserID("user-1"), fixedNow)

	if tm.ID() != "team-1" || tm.Name().Value() != "Stored" || tm.CreatedBy().Value() != "user-1" || !tm.CreatedAt().Equal(fixedNow) {
		t.Errorf("restored team = %s/%s/%s/%v", tm.ID(), tm.Name().Value(), tm.CreatedBy().Value(), tm.CreatedAt())
	}
}
