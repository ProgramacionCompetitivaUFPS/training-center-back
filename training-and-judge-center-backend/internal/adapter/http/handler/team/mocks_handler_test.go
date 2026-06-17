package team

import (
	"context"
	"time"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mockInvitationRepo ────────────────────────────────────────────────────────

type mockInvitationRepo struct {
	saveErr                error
	findByIDFn             func(id string) (*domainTeam.TeamInvitation, error)
	findByTeamAndInviteeFn func(teamID string, inviteeID domainShared.UserID) (*domainTeam.TeamInvitation, error)
	findByInviteeFn        func(inviteeID domainShared.UserID) ([]*domainTeam.TeamInvitation, error)
	deleteErr              error
}

func (m *mockInvitationRepo) Save(_ context.Context, _ *domainTeam.TeamInvitation) error {
	return m.saveErr
}
func (m *mockInvitationRepo) FindByID(_ context.Context, id string) (*domainTeam.TeamInvitation, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "invitation not found")
}
func (m *mockInvitationRepo) FindByTeamAndInvitee(_ context.Context, teamID string, inviteeID domainShared.UserID) (*domainTeam.TeamInvitation, error) {
	if m.findByTeamAndInviteeFn != nil {
		return m.findByTeamAndInviteeFn(teamID, inviteeID)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "invitation not found")
}
func (m *mockInvitationRepo) FindByInvitee(_ context.Context, inviteeID domainShared.UserID) ([]*domainTeam.TeamInvitation, error) {
	if m.findByInviteeFn != nil {
		return m.findByInviteeFn(inviteeID)
	}
	return []*domainTeam.TeamInvitation{}, nil
}
func (m *mockInvitationRepo) Delete(_ context.Context, _ string) error { return m.deleteErr }

// ── mockContestChecker ────────────────────────────────────────────────────────

type mockContestCheckerHandler struct {
	inActiveFn func(userID, teamID string) (bool, error)
}

func (m *mockContestCheckerHandler) IsUserInActiveContestForTeam(_ context.Context, userID, teamID string) (bool, error) {
	if m.inActiveFn != nil {
		return m.inActiveFn(userID, teamID)
	}
	return false, nil
}

// ── shared domain helpers ─────────────────────────────────────────────────────

func makeHandlerTeam(id, name, createdBy string, now time.Time) *domainTeam.Team {
	return domainTeam.RestoreTeam(id, domainTeam.RestoreTeamName(name), domainShared.RestoreUserID(createdBy), now)
}

func makeHandlerMember(id, teamID, userID string, now time.Time) *domainTeam.TeamMember {
	return domainTeam.RestoreTeamMember(id, teamID, domainShared.RestoreUserID(userID), now)
}
