package team

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListRegistrationsUseCase(
	groupChecker *mockGroupChecker,
	participRepo *mockTeamParticipRepo,
	teamRepo *mockTeamRepository,
) *ListTeamRegistrationsUseCase {
	return NewListTeamRegistrationsUseCase(groupChecker, participRepo, teamRepo, &mockUserProvider{})
}

func TestListTeamRegistrations_Success(t *testing.T) {
	part := domainContest.RestoreContestTeamParticipation(
		"part-001", testContestID, testTeamID, []string{memberA, memberB}, testNow,
	)
	uc := newListRegistrationsUseCase(
		&mockGroupChecker{},
		&mockTeamParticipRepo{listFn: func(_ string, _, _ int) ([]*domainContest.ContestTeamParticipation, int, error) {
			return []*domainContest.ContestTeamParticipation{part}, 1, nil
		}},
		teamRepoWithTeam(),
	)

	out, err := uc.Execute(ctx(), ListTeamRegistrationsInput{
		CurrentUser: asContestant(memberA),
		ContestID:   testContestID,
		GroupID:     testGroupID,
		Page:        1,
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Len(t, out.Items, 1)
	assert.Equal(t, testTeamID, out.Items[0].TeamID)
}

func TestListTeamRegistrations_NotGroupMember(t *testing.T) {
	uc := newListRegistrationsUseCase(
		&mockGroupChecker{fn: func(_ string, _ []string) (map[string]bool, error) {
			return map[string]bool{memberA: false}, nil
		}},
		&mockTeamParticipRepo{},
		teamRepoWithTeam(),
	)

	_, err := uc.Execute(ctx(), ListTeamRegistrationsInput{
		CurrentUser: asContestant(memberA),
		ContestID:   testContestID,
		GroupID:     testGroupID,
		Page:        1,
		Limit:       10,
	})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
}
