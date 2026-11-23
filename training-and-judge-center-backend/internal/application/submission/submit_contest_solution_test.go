package submission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var (
	contestStart = testNow.Add(-1 * time.Hour)
	contestEnd   = testNow.Add(1 * time.Hour)
)

func activeContestInfo() *ContestSubmissionInfo {
	return &ContestSubmissionInfo{
		ID:                testContestID,
		Name:              "ICPC 2026",
		GroupID:           testGroupID,
		StartTime:         contestStart,
		EndTime:           contestEnd,
		EnablePostContest: false,
		ProblemIDs:        []string{testProblemID},
	}
}

func contestInput() SubmitContestSolutionInput {
	return SubmitContestSolutionInput{
		CurrentUser: callerUser(),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		ProblemSlug: testProblemSlug,
		Language:    "cpp20",
		Compiler:    "g++",
		FileName:    "solution.cpp",
		FileData:    []byte("#include <bits/stdc++.h>"),
		SubmittedAt: testNow,
	}
}

func TestSubmitContestSolution_ActiveContest_PublishesWithContestPriority(t *testing.T) {
	q := &mockQueue{}
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) {
			return activeContestInfo(), nil
		}},
		&mockStandingIDResolver{},
		&mockGroupMembershipChecker{},
		publicProblem(),
		cleanRepo(),
		&mockSourceStorage{},
		q,
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	require.NoError(t, err)
	require.NotNil(t, q.lastPublished())
	assert.Equal(t, QueuePriorityContest, q.lastPublished().Priority)
}

func TestSubmitContestSolution_PostContest_PublishesWithPostContestPriority(t *testing.T) {
	q := &mockQueue{}
	finishedContest := &ContestSubmissionInfo{
		ID:                testContestID,
		Name:              "ICPC 2026",
		GroupID:           testGroupID,
		StartTime:         contestStart,
		EndTime:           testNow.Add(-30 * time.Minute),
		EnablePostContest: true,
		ProblemIDs:        []string{testProblemID},
	}
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) {
			return finishedContest, nil
		}},
		&mockStandingIDResolver{},
		&mockGroupMembershipChecker{},
		publicProblem(),
		cleanRepo(),
		&mockSourceStorage{},
		q,
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	require.NoError(t, err)
	require.NotNil(t, q.lastPublished())
	assert.Equal(t, QueuePriorityPostContest, q.lastPublished().Priority)
}

func TestSubmitContestSolution_BeforeContest_Returns400(t *testing.T) {
	futureContest := &ContestSubmissionInfo{
		ID:         testContestID,
		Name:       "ICPC 2026",
		GroupID:    testGroupID,
		StartTime:  testNow.Add(1 * time.Hour),
		EndTime:    testNow.Add(3 * time.Hour),
		ProblemIDs: []string{testProblemID},
	}
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) {
			return futureContest, nil
		}},
		&mockStandingIDResolver{},
		&mockGroupMembershipChecker{},
		publicProblem(),
		cleanRepo(),
		&mockSourceStorage{},
		&mockQueue{},
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeContestNotStarted, ae.Code)
}

func TestSubmitContestSolution_ContestFinished_PostDisabled_Returns400(t *testing.T) {
	finishedNoPost := &ContestSubmissionInfo{
		ID:                testContestID,
		Name:              "ICPC 2026",
		GroupID:           testGroupID,
		StartTime:         contestStart,
		EndTime:           testNow.Add(-30 * time.Minute),
		EnablePostContest: false,
		ProblemIDs:        []string{testProblemID},
	}
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) {
			return finishedNoPost, nil
		}},
		&mockStandingIDResolver{},
		&mockGroupMembershipChecker{},
		publicProblem(),
		cleanRepo(),
		&mockSourceStorage{},
		&mockQueue{},
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeContestFinished, ae.Code)
}

func finishedContestInfo(enablePost bool) *ContestSubmissionInfo {
	return &ContestSubmissionInfo{
		ID:                testContestID,
		Name:              "ICPC 2026",
		GroupID:           testGroupID,
		StartTime:         contestStart,
		EndTime:           testNow.Add(-30 * time.Minute),
		EnablePostContest: enablePost,
		ProblemIDs:        []string{testProblemID},
	}
}

func notRegistered() *mockStandingIDResolver {
	return &mockStandingIDResolver{fn: func(_, _ string) (string, bool, error) { return "", false, nil }}
}

func submitAsUnregistered(contest *ContestSubmissionInfo, members *mockGroupMembershipChecker, repo *mockSubmissionRepo, q *mockQueue) error {
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) { return contest, nil }},
		notRegistered(),
		members,
		publicProblem(),
		repo,
		&mockSourceStorage{},
		q,
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	return err
}

func TestSubmitContestSolution_Unregistered_PostContestEnabled_GroupMember_UsesOwnUserAsStanding(t *testing.T) {
	repo, q := cleanRepo(), &mockQueue{}
	err := submitAsUnregistered(finishedContestInfo(true), &mockGroupMembershipChecker{}, repo, q)
	require.NoError(t, err)
	require.NotNil(t, repo.saved)
	require.NotNil(t, repo.saved.StandingID())
	assert.Equal(t, testUserID, *repo.saved.StandingID())
	assert.Equal(t, QueuePriorityPostContest, q.lastPublished().Priority)
}

func TestSubmitContestSolution_Unregistered_PostContestEnabled_NotGroupMember_Returns403(t *testing.T) {
	members := &mockGroupMembershipChecker{fn: func(_, _ string) (bool, error) { return false, nil }}
	err := submitAsUnregistered(finishedContestInfo(true), members, cleanRepo(), &mockQueue{})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, ErrCodeNotRegistered, ae.Code)
}

func TestSubmitContestSolution_Unregistered_PostContestDisabled_Returns403(t *testing.T) {
	err := submitAsUnregistered(finishedContestInfo(false), &mockGroupMembershipChecker{}, cleanRepo(), &mockQueue{})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, ErrCodeNotRegistered, ae.Code)
}

func TestSubmitContestSolution_Unregistered_DuringContest_Returns403(t *testing.T) {
	err := submitAsUnregistered(activeContestInfo(), &mockGroupMembershipChecker{}, cleanRepo(), &mockQueue{})
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, ErrCodeNotRegistered, ae.Code)
}

func TestSubmitContestSolution_Registered_PostContest_KeepsResolvedStanding(t *testing.T) {
	repo := cleanRepo()
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) { return finishedContestInfo(true), nil }},
		&mockStandingIDResolver{fn: func(_, _ string) (string, bool, error) { return "team-001", true, nil }},
		&mockGroupMembershipChecker{fn: func(_, _ string) (bool, error) { return false, nil }},
		publicProblem(),
		repo,
		&mockSourceStorage{},
		&mockQueue{},
		maxFileSize,
		rateLimitSeconds,
	)
	_, err := uc.Execute(ctx(), contestInput())
	require.NoError(t, err)
	assert.Equal(t, "team-001", *repo.saved.StandingID())
}
