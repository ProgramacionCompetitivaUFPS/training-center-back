package team

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUpdateUseCase(
	memberRepo *mockMemberRepository,
	teamRepo *mockTeamRepository,
	contestProv *mockContestProvider,
	participRepo *mockTeamParticipRepo,
	indivChecker *mockIndivChecker,
	groupChecker *mockGroupChecker,
) *UpdateTeamRegistrationUseCase {
	return NewUpdateTeamRegistrationUseCase(
		memberRepo, teamRepo, contestProv, participRepo,
		indivChecker, groupChecker, &mockUserProvider{}, &mockTxManager{},
	)
}

func defaultUpdateInput() UpdateTeamRegistrationInput {
	return UpdateTeamRegistrationInput{
		CurrentUser:     asContestant(memberA),
		ContestID:       testContestID,
		TeamID:          testTeamID,
		SelectedMembers: []string{memberA, memberC},
	}
}

func TestUpdateTeamRegistration_Success(t *testing.T) {
	uc := newUpdateUseCase(
		memberRepoWithUsers(memberA, memberB, memberC),
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	out, err := uc.Execute(ctx(), defaultUpdateInput())
	require.NoError(t, err)
	assert.Equal(t, testContestID, out.ContestID)
	assert.Equal(t, testTeamID, out.TeamID)
	assert.Len(t, out.SelectedMembers, 2)
}

func TestUpdateTeamRegistration_ContestAlreadyStarted(t *testing.T) {
	uc := newUpdateUseCase(
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

	_, err := uc.Execute(ctx(), defaultUpdateInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, domainContest.ErrCodeContestAlreadyStarted, ae.Code)
}

func TestUpdateTeamRegistration_RequesterNotTeamMember(t *testing.T) {
	uc := newUpdateUseCase(
		&mockMemberRepository{},
		teamRepoWithTeam(),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
		&mockIndivChecker{},
		&mockGroupChecker{},
	)

	_, err := uc.Execute(ctx(), defaultUpdateInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, domainTeam.ErrCodeNotTeamMember, ae.Code)
}
