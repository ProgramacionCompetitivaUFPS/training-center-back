package team

import (
	"context"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
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

// ── mockContestProvider ───────────────────────────────────────────────────────

type mockContestProvider struct {
	getInfoFn func(contestID string) (*appTeam.ContestInfo, error)
}

func (m *mockContestProvider) GetContestInfo(_ context.Context, contestID string) (*appTeam.ContestInfo, error) {
	if m.getInfoFn != nil {
		return m.getInfoFn(contestID)
	}
	return &appTeam.ContestInfo{
		ID:                contestID,
		GroupID:           "g1",
		ParticipationMode: "TEAM",
		TeamSizeMin:       1,
		TeamSizeMax:       3,
		Status:            "SCHEDULED",
	}, nil
}

// ── mockTeamParticipRepo ──────────────────────────────────────────────────────

type mockTeamParticipRepo struct {
	saveFn                      func(*domainContest.ContestTeamParticipation) error
	findByContestAndTeamFn      func(contestID, teamID string) (*domainContest.ContestTeamParticipation, error)
	updateFn                    func(*domainContest.ContestTeamParticipation) error
	deleteFn                    func(contestID, teamID string) error
	listFn                      func(contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error)
	areUsersSelectedInAnyTeamFn func(contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error)
}

func (m *mockTeamParticipRepo) Save(_ context.Context, p *domainContest.ContestTeamParticipation) error {
	if m.saveFn != nil {
		return m.saveFn(p)
	}
	return nil
}
func (m *mockTeamParticipRepo) FindByContestAndTeam(_ context.Context, contestID, teamID string) (*domainContest.ContestTeamParticipation, error) {
	if m.findByContestAndTeamFn != nil {
		return m.findByContestAndTeamFn(contestID, teamID)
	}
	return domainContest.RestoreContestTeamParticipation("p1", contestID, teamID, []string{"u1"}, time.Now()), nil
}
func (m *mockTeamParticipRepo) Update(_ context.Context, p *domainContest.ContestTeamParticipation) error {
	if m.updateFn != nil {
		return m.updateFn(p)
	}
	return nil
}
func (m *mockTeamParticipRepo) Delete(_ context.Context, contestID, teamID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(contestID, teamID)
	}
	return nil
}
func (m *mockTeamParticipRepo) List(_ context.Context, contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error) {
	if m.listFn != nil {
		return m.listFn(contestID, page, limit)
	}
	return []*domainContest.ContestTeamParticipation{}, 0, nil
}
func (m *mockTeamParticipRepo) AreUsersSelectedInAnyTeam(_ context.Context, contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error) {
	if m.areUsersSelectedInAnyTeamFn != nil {
		return m.areUsersSelectedInAnyTeamFn(contestID, userIDs, excludeTeamID)
	}
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	return result, nil
}

// ── mockIndivChecker ──────────────────────────────────────────────────────────

type mockIndivChecker struct {
	fn func(contestID string, userIDs []string) (map[string]bool, error)
}

func (m *mockIndivChecker) AreUsersRegisteredIndividually(_ context.Context, contestID string, userIDs []string) (map[string]bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userIDs)
	}
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	return result, nil
}

// ── mockGroupChecker ──────────────────────────────────────────────────────────

type mockGroupChecker struct {
	fn func(groupID string, userIDs []string) (map[string]bool, error)
}

func (m *mockGroupChecker) AreUsersGroupMembers(_ context.Context, groupID string, userIDs []string) (map[string]bool, error) {
	if m.fn != nil {
		return m.fn(groupID, userIDs)
	}
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = true
	}
	return result, nil
}

// ── mockScheduledParticipationCleanerHandler ──────────────────────────────────

type mockScheduledParticipationCleanerHandler struct{}

func (m *mockScheduledParticipationCleanerHandler) RemoveUserFromScheduledParticipations(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
