package team

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── time fixture ─────────────────────────────────────────────────────────────

var testNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

// ── test constants ────────────────────────────────────────────────────────────

const (
	memberA   = "aaaaaaaa-0000-0000-0000-000000000001"
	memberB   = "aaaaaaaa-0000-0000-0000-000000000002"
	memberC   = "aaaaaaaa-0000-0000-0000-000000000003"
	strangerID = "aaaaaaaa-0000-0000-0000-000000000099"
	testTeamID    = "bbbbbbbb-0000-0000-0000-000000000001"
	testContestID = "cccccccc-0000-0000-0000-000000000001"
	testGroupID   = "dddddddd-0000-0000-0000-000000000001"
)

// ── CurrentUser helpers ──────────────────────────────────────────────────────

var (
	asAdmin      = testutil.AsAdmin
	asCoach      = testutil.AsCoach
	asContestant = testutil.AsContestant
)

func ctx() context.Context { return context.Background() }

// ── mockTeamRepository ───────────────────────────────────────────────────────

type mockTeamRepository struct {
	saveErr          error
	existsByNameFn   func(name domainTeam.TeamName) (bool, error)
	findByIDFn       func(id string) (*domainTeam.Team, error)
	findByIDsFn      func(ids []string) ([]*domainTeam.Team, error)
}

func (m *mockTeamRepository) Save(_ context.Context, _ *domainTeam.Team) error {
	return m.saveErr
}

func (m *mockTeamRepository) FindByID(_ context.Context, id string) (*domainTeam.Team, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
}

func (m *mockTeamRepository) FindByIDs(_ context.Context, ids []string) ([]*domainTeam.Team, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ids)
	}
	return []*domainTeam.Team{}, nil
}

func (m *mockTeamRepository) ExistsByName(_ context.Context, name domainTeam.TeamName) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(name)
	}
	return false, nil
}

// ── mockMemberRepository ─────────────────────────────────────────────────────

type mockMemberRepository struct {
	saveErr              error
	findByTeamFn         func(teamID string) ([]*domainTeam.TeamMember, error)
	findByTeamAndUserFn  func(teamID string, userID shared.UserID) (*domainTeam.TeamMember, error)
	findByUserFn         func(userID shared.UserID) ([]*domainTeam.TeamMember, error)
	bulkCountFn          func(teamIDs []string) (map[string]int, error)
	deleteByTeamAndUserFn func(teamID string, userID shared.UserID) error
}

func (m *mockMemberRepository) Save(_ context.Context, _ *domainTeam.TeamMember) error {
	return m.saveErr
}

func (m *mockMemberRepository) FindByTeam(_ context.Context, teamID string) ([]*domainTeam.TeamMember, error) {
	if m.findByTeamFn != nil {
		return m.findByTeamFn(teamID)
	}
	return []*domainTeam.TeamMember{}, nil
}

func (m *mockMemberRepository) FindByTeamAndUser(_ context.Context, teamID string, userID shared.UserID) (*domainTeam.TeamMember, error) {
	if m.findByTeamAndUserFn != nil {
		return m.findByTeamAndUserFn(teamID, userID)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
}

func (m *mockMemberRepository) FindByUser(_ context.Context, userID shared.UserID) ([]*domainTeam.TeamMember, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(userID)
	}
	return []*domainTeam.TeamMember{}, nil
}

func (m *mockMemberRepository) BulkCountByTeams(_ context.Context, teamIDs []string) (map[string]int, error) {
	if m.bulkCountFn != nil {
		return m.bulkCountFn(teamIDs)
	}
	return map[string]int{}, nil
}

func (m *mockMemberRepository) DeleteByTeamAndUser(_ context.Context, teamID string, userID shared.UserID) error {
	if m.deleteByTeamAndUserFn != nil {
		return m.deleteByTeamAndUserFn(teamID, userID)
	}
	return nil
}

// ── mockUserProvider ─────────────────────────────────────────────────────────

type mockUserProvider struct {
	displayFn       func(userID string) (*UserDisplay, error)
	displaysFn      func(userIDs []string) (map[string]*UserDisplay, error)
	findByNicknameFn func(nickname string) (*UserDisplay, error)
}

func (m *mockUserProvider) GetDisplay(_ context.Context, userID string) (*UserDisplay, error) {
	if m.displayFn != nil {
		return m.displayFn(userID)
	}
	return &UserDisplay{Nickname: "testuser"}, nil
}

func (m *mockUserProvider) GetDisplays(_ context.Context, userIDs []string) (map[string]*UserDisplay, error) {
	if m.displaysFn != nil {
		return m.displaysFn(userIDs)
	}
	result := make(map[string]*UserDisplay, len(userIDs))
	for _, id := range userIDs {
		result[id] = &UserDisplay{ID: id, Nickname: "testuser"}
	}
	return result, nil
}

func (m *mockUserProvider) FindByNickname(_ context.Context, nickname string) (*UserDisplay, error) {
	if m.findByNicknameFn != nil {
		return m.findByNicknameFn(nickname)
	}
	return &UserDisplay{ID: "default-user-id", Nickname: nickname}, nil
}

// ── mockInvitationRepository ─────────────────────────────────────────────────

type mockInvitationRepository struct {
	saveErr                 error
	findByIDFn              func(id string) (*domainTeam.TeamInvitation, error)
	findByTeamAndInviteeFn  func(teamID string, inviteeID shared.UserID) (*domainTeam.TeamInvitation, error)
	findByInviteeFn         func(inviteeID shared.UserID) ([]*domainTeam.TeamInvitation, error)
	deleteFn                func(id string) error
	deleteErr               error
}

func (m *mockInvitationRepository) Save(_ context.Context, _ *domainTeam.TeamInvitation) error {
	return m.saveErr
}

func (m *mockInvitationRepository) FindByID(_ context.Context, id string) (*domainTeam.TeamInvitation, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "invitation not found")
}

func (m *mockInvitationRepository) FindByTeamAndInvitee(_ context.Context, teamID string, inviteeID shared.UserID) (*domainTeam.TeamInvitation, error) {
	if m.findByTeamAndInviteeFn != nil {
		return m.findByTeamAndInviteeFn(teamID, inviteeID)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "invitation not found")
}

func (m *mockInvitationRepository) FindByInvitee(_ context.Context, inviteeID shared.UserID) ([]*domainTeam.TeamInvitation, error) {
	if m.findByInviteeFn != nil {
		return m.findByInviteeFn(inviteeID)
	}
	return []*domainTeam.TeamInvitation{}, nil
}

func (m *mockInvitationRepository) Delete(_ context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return m.deleteErr
}

// ── mockContestParticipationChecker ──────────────────────────────────────────

type mockContestChecker struct {
	inActiveFn func(userID, teamID string) (bool, error)
}

func (m *mockContestChecker) IsUserInActiveContestForTeam(_ context.Context, userID, teamID string) (bool, error) {
	if m.inActiveFn != nil {
		return m.inActiveFn(userID, teamID)
	}
	return false, nil
}

// ── mockTxManager ────────────────────────────────────────────────────────────

type mockTxManager struct{}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ── mockContestProvider ───────────────────────────────────────────────────────

type mockContestProvider struct {
	infoFn func(contestID string) (*ContestInfo, error)
}

func (m *mockContestProvider) GetContestInfo(_ context.Context, contestID string) (*ContestInfo, error) {
	if m.infoFn != nil {
		return m.infoFn(contestID)
	}
	return &ContestInfo{
		ID:                contestID,
		GroupID:           testGroupID,
		ParticipationMode: "TEAM",
		TeamSizeMin:       2,
		TeamSizeMax:       3,
		Status:            "SCHEDULED",
	}, nil
}

// ── mockIndividualRegistrationChecker ─────────────────────────────────────────

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

// ── mockGroupMemberChecker ────────────────────────────────────────────────────

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

// ── mockTeamParticipationRepository ──────────────────────────────────────────

type mockTeamParticipRepo struct {
	saveFn                    func(p *domainContest.ContestTeamParticipation) error
	findByContestAndTeamFn    func(contestID, teamID string) (*domainContest.ContestTeamParticipation, error)
	updateFn                  func(p *domainContest.ContestTeamParticipation) error
	deleteFn                  func(contestID, teamID string) error
	listFn                    func(contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error)
	areUsersSelectedFn        func(contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error)
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
	return domainContest.RestoreContestTeamParticipation("part-001", contestID, teamID,
		[]string{memberA, memberB}, testNow), nil
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
	if m.areUsersSelectedFn != nil {
		return m.areUsersSelectedFn(contestID, userIDs, excludeTeamID)
	}
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	return result, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newCreateTeamUseCase(teamRepo *mockTeamRepository, memberRepo *mockMemberRepository) *CreateTeamUseCase {
	if memberRepo == nil {
		memberRepo = &mockMemberRepository{}
	}
	return NewCreateTeamUseCase(teamRepo, memberRepo, &mockUserProvider{}, &mockTxManager{})
}

func validInput(cu appshared.CurrentUser) CreateTeamInput {
	return CreateTeamInput{Name: "Alpha Team", CurrentUser: cu}
}

// ── mockScheduledParticipationCleaner ─────────────────────────────────────────

type mockScheduledParticipationCleaner struct {
	removedCount int
	err          error
	called       bool
}

func (m *mockScheduledParticipationCleaner) RemoveUserFromScheduledParticipations(_ context.Context, _, _ string) (int, error) {
	m.called = true
	return m.removedCount, m.err
}
