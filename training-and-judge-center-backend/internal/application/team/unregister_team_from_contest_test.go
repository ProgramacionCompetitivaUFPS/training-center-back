package team

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUnregisterUseCase(
	memberRepo *mockMemberRepository,
	contestProv *mockContestProvider,
	participRepo *mockTeamParticipRepo,
) *UnregisterTeamFromContestUseCase {
	return NewUnregisterTeamFromContestUseCase(memberRepo, contestProv, participRepo)
}

func TestUnregisterTeamFromContest_Success(t *testing.T) {
	uc := newUnregisterUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{},
		&mockTeamParticipRepo{},
	)

	err := uc.Execute(ctx(), UnregisterTeamFromContestInput{
		CurrentUser: asContestant(memberA),
		ContestID:   testContestID,
		TeamID:      testTeamID,
	})
	require.NoError(t, err)
}

func TestUnregisterTeamFromContest_ContestAlreadyStarted(t *testing.T) {
	uc := newUnregisterUseCase(
		memberRepoWithUsers(memberA),
		&mockContestProvider{infoFn: func(_ string) (*ContestInfo, error) {
			return &ContestInfo{Status: "ACTIVE", ParticipationMode: "TEAM", TeamSizeMin: 2, TeamSizeMax: 3}, nil
		}},
		&mockTeamParticipRepo{},
	)

	err := uc.Execute(ctx(), UnregisterTeamFromContestInput{
		CurrentUser: asContestant(memberA),
		ContestID:   testContestID,
		TeamID:      testTeamID,
	})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, domainContest.ErrCodeContestAlreadyStarted, ae.Code)
}

func TestUnregisterTeamFromContest_RequesterNotTeamMember(t *testing.T) {
	uc := newUnregisterUseCase(
		&mockMemberRepository{},
		&mockContestProvider{},
		&mockTeamParticipRepo{},
	)

	err := uc.Execute(ctx(), UnregisterTeamFromContestInput{
		CurrentUser: asContestant(memberA),
		ContestID:   testContestID,
		TeamID:      testTeamID,
	})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, domainTeam.ErrCodeNotTeamMember, ae.Code)
}
