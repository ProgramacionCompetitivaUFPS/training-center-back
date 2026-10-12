package team

import (
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newEligibilityUseCase(
	memberRepo *mockMemberRepository,
	contestProv *mockContestProvider,
	participRepo *mockTeamParticipRepo,
	indivChecker *mockIndivChecker,
	groupChecker *mockGroupChecker,
) *GetTeamRegistrationEligibilityUseCase {
	if participRepo.findByContestAndTeamFn == nil {
		participRepo.findByContestAndTeamFn = func(_, _ string) (*domainContest.ContestTeamParticipation, error) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeTeamRegistrationNotFound, "team registration not found")
		}
	}
	return NewGetTeamRegistrationEligibilityUseCase(
		memberRepo, contestProv, participRepo, indivChecker, groupChecker, &mockUserProvider{},
	)
}

func defaultEligibilityInput() GetTeamRegistrationEligibilityInput {
	return GetTeamRegistrationEligibilityInput{
		CurrentUser: asContestant(memberA),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		TeamID:      testTeamID,
	}
}

func TestGetTeamRegistrationEligibility_AllEligible(t *testing.T) {
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{}, &mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
	)

	out, err := uc.Execute(ctx(), defaultEligibilityInput())
	require.NoError(t, err)
	assert.False(t, out.TeamAlreadyRegistered)
	require.Len(t, out.Members, 2)
	for _, m := range out.Members {
		assert.True(t, m.Eligible)
		assert.Empty(t, m.Reason)
		assert.Equal(t, "testuser", m.Nickname)
	}
}

func TestGetTeamRegistrationEligibility_ReportsReasonPerMember(t *testing.T) {
	groupChecker := &mockGroupChecker{
		fn: func(_ string, userIDs []string) (map[string]bool, error) {
			return map[string]bool{memberA: true, memberB: false, memberC: true}, nil
		},
	}
	indivChecker := &mockIndivChecker{
		fn: func(_ string, _ []string) (map[string]bool, error) {
			return map[string]bool{memberC: true}, nil
		},
	}
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB, memberC),
		&mockContestProvider{}, &mockTeamParticipRepo{}, indivChecker, groupChecker,
	)

	out, err := uc.Execute(ctx(), defaultEligibilityInput())
	require.NoError(t, err)
	require.Len(t, out.Members, 3)
	assert.True(t, out.Members[0].Eligible)
	assert.Equal(t, ErrCodeMemberNotInGroup, out.Members[1].Reason)
	assert.False(t, out.Members[1].Eligible)
	assert.Equal(t, ErrCodeMemberAlreadyRegistered, out.Members[2].Reason)
	assert.False(t, out.Members[2].Eligible)
}

func TestGetTeamRegistrationEligibility_MemberInAnotherTeam(t *testing.T) {
	participRepo := &mockTeamParticipRepo{
		areUsersSelectedFn: func(_ string, _ []string, _ string) (map[string]bool, error) {
			return map[string]bool{memberB: true}, nil
		},
	}
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{}, participRepo, &mockIndivChecker{}, &mockGroupChecker{},
	)

	out, err := uc.Execute(ctx(), defaultEligibilityInput())
	require.NoError(t, err)
	assert.True(t, out.Members[0].Eligible)
	assert.Equal(t, ErrCodeMemberInAnotherTeam, out.Members[1].Reason)
}

func TestGetTeamRegistrationEligibility_NotTeamMemberIsForbidden(t *testing.T) {
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{}, &mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
	)
	in := defaultEligibilityInput()
	in.CurrentUser = asContestant(strangerID)

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, domainTeam.ErrCodeNotTeamMember, ae.Code)
}

func TestGetTeamRegistrationEligibility_ContestFromAnotherGroupIsNotFound(t *testing.T) {
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{}, &mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
	)
	in := defaultEligibilityInput()
	in.GroupID = "other-group"

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.KindNotFound, ae.Kind)
	assert.Equal(t, ErrCodeContestNotFound, ae.Code)
}

func TestGetTeamRegistrationEligibility_TeamAlreadyRegistered(t *testing.T) {
	var excluded string
	participRepo := &mockTeamParticipRepo{
		findByContestAndTeamFn: func(contestID, teamID string) (*domainContest.ContestTeamParticipation, error) {
			return domainContest.RestoreContestTeamParticipation("part-001", contestID, teamID, []string{memberB}, testNow), nil
		},
		areUsersSelectedFn: func(_ string, _ []string, excludeTeamID string) (map[string]bool, error) {
			excluded = excludeTeamID
			return map[string]bool{}, nil
		},
	}
	uc := newEligibilityUseCase(
		memberRepoWithUsers(memberA, memberB),
		&mockContestProvider{}, participRepo, &mockIndivChecker{}, &mockGroupChecker{},
	)

	out, err := uc.Execute(ctx(), defaultEligibilityInput())
	require.NoError(t, err)
	assert.True(t, out.TeamAlreadyRegistered)
	assert.Equal(t, testTeamID, excluded)
	assert.True(t, out.Members[1].Eligible)
}
