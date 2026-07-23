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
