package team

import (
	"context"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
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
