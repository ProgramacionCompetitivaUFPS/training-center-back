package team_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/team"
)

func TestNewTeamMember_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	m, err := team.NewTeamMember("member-1", "team-1", shared.RestoreUserID("user-1"), localNow)

	if err != nil {
		t.Fatalf("NewTeamMember: %v", err)
	}
	if m.ID() != "member-1" || m.TeamID() != "team-1" || m.UserID().Value() != "user-1" {
		t.Errorf("fields = %s/%s/%s, want member-1/team-1/user-1", m.ID(), m.TeamID(), m.UserID().Value())
	}
	assertInUTC(t, "JoinedAt", m.JoinedAt(), utcOfNow)
}

func TestNewTeamMember_MissingData_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name       string
		id, teamID string
		userID     shared.UserID
	}{
		{"an empty id", "", "team-1", shared.RestoreUserID("user-1")},
		{"an empty team id", "member-1", "", shared.RestoreUserID("user-1")},
		{"an empty user id", "member-1", "team-1", emptyUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := team.NewTeamMember(tt.id, tt.teamID, tt.userID, fixedNow)

			assertInternalError(t, tt.name, err)
			if m != nil {
				t.Errorf("expected no member, got %+v", m)
			}
		})
	}
}

func TestRestoreTeamMember_KeepsTheStoredValues(t *testing.T) {
	m := team.RestoreTeamMember("member-1", "team-1", shared.RestoreUserID("user-1"), fixedNow)

	if m.ID() != "member-1" || m.TeamID() != "team-1" || m.UserID().Value() != "user-1" || !m.JoinedAt().Equal(fixedNow) {
		t.Errorf("restored member = %s/%s/%s/%v", m.ID(), m.TeamID(), m.UserID().Value(), m.JoinedAt())
	}
}
