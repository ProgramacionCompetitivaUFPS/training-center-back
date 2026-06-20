package team

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRegisterUseCase(
	memberRepo *mockMemberRepository,
	teamRepo *mockTeamRepository,
	contestProv *mockContestProvider,
	participRepo *mockTeamParticipRepo,
	indivChecker *mockIndivChecker,
	groupChecker *mockGroupChecker,
) *RegisterTeamToContestUseCase {
	return NewRegisterTeamToContestUseCase(
		memberRepo, teamRepo, contestProv, participRepo,
		indivChecker, groupChecker, &mockUserProvider{}, &mockTxManager{},
	)
}

func defaultRegisterInput() RegisterTeamToContestInput {
	return RegisterTeamToContestInput{
		CurrentUser:     asContestant(memberA),
		ContestID:       testContestID,
		TeamID:          testTeamID,
		SelectedMembers: []string{memberA, memberB},
	}
}

func memberRepoWithUsers(ids ...string) *mockMemberRepository {
	members := make([]*domainTeam.TeamMember, len(ids))
	for i, id := range ids {
		members[i] = domainTeam.RestoreTeamMember("m-"+id, testTeamID, shared.RestoreUserID(id), testNow)
	}
	return &mockMemberRepository{
		findByTeamAndUserFn: func(teamID string, userID shared.UserID) (*domainTeam.TeamMember, error) {
			for _, uid := range ids {
				if uid == userID.Value() {
					return domainTeam.RestoreTeamMember("m-"+uid, teamID, shared.RestoreUserID(uid), testNow), nil
				}
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
		findByTeamFn: func(teamID string) ([]*domainTeam.TeamMember, error) {
			return members, nil
		},
	}
}

func teamRepoWithTeam() *mockTeamRepository {
	t := domainTeam.RestoreTeam(testTeamID, domainTeam.RestoreTeamName("Alpha"), shared.RestoreUserID(memberA), testNow)
	return &mockTeamRepository{
		findByIDFn: func(id string) (*domainTeam.Team, error) {
			return t, nil
		},
	}
}

func TestRegisterTeamToContest_Success(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB, memberC),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	out, err := uc.Execute(ctx(), defaultRegisterInput())
	require.NoError(t, err)
	assert.Equal(t, testContestID, out.ContestID)
	assert.Equal(t, testTeamID, out.TeamID)
	assert.Len(t, out.SelectedMembers, 2)
}

func TestRegisterTeamToContest_RequesterNotTeamMember(t *testing.T) {
	uc := newRegisterUseCase(
		&mockMemberRepository{}, // default returns not found
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, domainTeam.ErrCodeNotTeamMember, ae.Code)
}

func TestRegisterTeamToContest_TeamsNotAllowed(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{infoFn: func(_ string) (*ContestInfo, error) {
			return &ContestInfo{
				ID: testContestID, GroupID: testGroupID,
				ParticipationMode: "INDIVIDUAL", TeamSizeMin: 2, TeamSizeMax: 3,
				Status: "SCHEDULED",
			}, nil
		}},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeTeamsNotAllowed, ae.Code)
}

func TestRegisterTeamToContest_ContestNotScheduled(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{infoFn: func(_ string) (*ContestInfo, error) {
			return &ContestInfo{
				ID: testContestID, GroupID: testGroupID,
				ParticipationMode: "TEAM", TeamSizeMin: 2, TeamSizeMax: 3,
				Status: "ACTIVE",
			}, nil
		}},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, ErrCodeContestNotOpenForRegistration, ae.Code)
}

func TestRegisterTeamToContest_InvalidSelectedMember(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB), // memberC not in team
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	in := defaultRegisterInput()
	in.SelectedMembers = []string{memberA, memberC} // memberC is not in team
	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeInvalidSelectedMember, ae.Code)
}

func TestRegisterTeamToContest_InvalidTeamSize(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB, memberC),
		teamRepoWithTeam(),
		&mockContestProvider{infoFn: func(_ string) (*ContestInfo, error) {
			return &ContestInfo{
				ID: testContestID, GroupID: testGroupID,
				ParticipationMode: "TEAM", TeamSizeMin: 3, TeamSizeMax: 3,
				Status: "SCHEDULED",
			}, nil
		}},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	// Only 2 selected but min is 3
	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeInvalidTeamSize, ae.Code)
}

func TestRegisterTeamToContest_MemberNotInGroup(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{fn: func(groupID string, userIDs []string) (map[string]bool, error) {
			return map[string]bool{memberA: true, memberB: false}, nil
		}},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, ErrCodeMemberNotInGroup, ae.Code)
}

func TestRegisterTeamToContest_MemberAlreadyRegisteredIndividually(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{fn: func(contestID string, userIDs []string) (map[string]bool, error) {
			return map[string]bool{memberA: true, memberB: false}, nil
		}},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, ErrCodeMemberAlreadyRegistered, ae.Code)
}

func TestRegisterTeamToContest_MemberInAnotherTeam(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{areUsersSelectedFn: func(contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error) {
			return map[string]bool{memberA: false, memberB: true}, nil
		}},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, ErrCodeMemberInAnotherTeam, ae.Code)
}

func TestRegisterTeamToContest_TeamAlreadyRegistered(t *testing.T) {
	uc := newRegisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{saveFn: func(p *domainContest.ContestTeamParticipation) error {
			return apperror.NewConflict(domainContest.ErrCodeTeamAlreadyRegistered, "team already registered")
		}},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultRegisterInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, domainContest.ErrCodeTeamAlreadyRegistered, ae.Code)
}
