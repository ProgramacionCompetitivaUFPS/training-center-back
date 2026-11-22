package team_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/team"
)

func TestNewTeamInvitation_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	inv, err := team.NewTeamInvitation("inv-1", "team-1", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("inviter-1"), localNow)

	if err != nil {
		t.Fatalf("NewTeamInvitation: %v", err)
	}
	if inv.ID() != "inv-1" || inv.TeamID() != "team-1" || inv.InviteeID().Value() != "invitee-1" || inv.InvitedBy().Value() != "inviter-1" {
		t.Errorf("fields = %s/%s/%s/%s, want inv-1/team-1/invitee-1/inviter-1", inv.ID(), inv.TeamID(), inv.InviteeID().Value(), inv.InvitedBy().Value())
	}
	assertInUTC(t, "CreatedAt", inv.CreatedAt(), utcOfNow)
}

func TestNewTeamInvitation_MissingData_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name                 string
		id, teamID           string
		inviteeID, invitedBy shared.UserID
	}{
		{"an empty id", "", "team-1", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("inviter-1")},
		{"an empty team id", "inv-1", "", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("inviter-1")},
		{"an empty invitee", "inv-1", "team-1", emptyUser, shared.RestoreUserID("inviter-1")},
		{"an empty inviter", "inv-1", "team-1", shared.RestoreUserID("invitee-1"), emptyUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := team.NewTeamInvitation(tt.id, tt.teamID, tt.inviteeID, tt.invitedBy, fixedNow)

			assertInternalError(t, tt.name, err)
			if inv != nil {
				t.Errorf("expected no invitation, got %+v", inv)
			}
		})
	}
}

func TestRestoreTeamInvitation_KeepsTheStoredValues(t *testing.T) {
	inv := team.RestoreTeamInvitation("inv-1", "team-1", shared.RestoreUserID("invitee-1"), shared.RestoreUserID("inviter-1"), fixedNow)

	if inv.ID() != "inv-1" || inv.TeamID() != "team-1" || inv.InviteeID().Value() != "invitee-1" || inv.InvitedBy().Value() != "inviter-1" || !inv.CreatedAt().Equal(fixedNow) {
		t.Errorf("restored invitation = %s/%s/%s/%s/%v", inv.ID(), inv.TeamID(), inv.InviteeID().Value(), inv.InvitedBy().Value(), inv.CreatedAt())
	}
}
