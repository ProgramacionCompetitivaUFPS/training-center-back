package submission

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── time / constants ──────────────────────────────────────────────────────────

var testNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testUserID    = "aaaaaaaa-0000-0000-0000-000000000001"
	testProblemID = "bbbbbbbb-0000-0000-0000-000000000001"
	testProblemSlug = "sum-of-two-numbers"

	maxFileSize      = int64(1 << 20) // 1 MiB
	rateLimitSeconds = 1
)

var (
	asAdmin      = testutil.AsAdmin
	asContestant = testutil.AsContestant
)

func callerUser() appshared.CurrentUser {
	return asContestant(testUserID)
}

// ── mockProblemProvider ───────────────────────────────────────────────────────

type mockProblemProvider struct {
	fn func(slug string) (*ProblemInfo, error)
}

func (m *mockProblemProvider) GetProblemBySlug(_ context.Context, slug string) (*ProblemInfo, error) {
	if m.fn != nil {
		return m.fn(slug)
	}
	return &ProblemInfo{
		ID:           testProblemID,
		Slug:         testProblemSlug,
		Title:        "Sum of Two Numbers",
		IsPublished:  true,
		IsPublic:     true,
		ModifierIDs:  []string{},
		HasTestCases: true,
	}, nil
}

func publicProblem() *mockProblemProvider { return &mockProblemProvider{} }

func draftProblem() *mockProblemProvider {
	return &mockProblemProvider{fn: func(_ string) (*ProblemInfo, error) {
		return &ProblemInfo{ID: testProblemID, Slug: testProblemSlug, Title: "T", IsPublished: false, IsPublic: true}, nil
	}}
}

func privateProblem(modifierIDs ...string) *mockProblemProvider {
	return &mockProblemProvider{fn: func(_ string) (*ProblemInfo, error) {
		return &ProblemInfo{
			ID: testProblemID, Slug: testProblemSlug, Title: "T",
			IsPublished: true, IsPublic: false, ModifierIDs: modifierIDs,
		}, nil
	}}
}

// ── mockSubmissionRepository ──────────────────────────────────────────────────

type mockSubmissionRepo struct {
	saveErr      error
	existsDupFn  func(hash, userID, problemID string) (bool, error)
	lastFn       func(userID, problemID string) (*domainSubmission.Submission, error)
}

func (m *mockSubmissionRepo) Save(_ context.Context, _ *domainSubmission.Submission) error {
	return m.saveErr
}

func (m *mockSubmissionRepo) FindByID(_ context.Context, _ string) (*domainSubmission.Submission, error) {
	return nil, apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "not found")
}

func (m *mockSubmissionRepo) List(_ context.Context, _ domainSubmission.ListFilters) ([]*domainSubmission.Submission, int, error) {
	return nil, 0, nil
}

func (m *mockSubmissionRepo) FindLastByUserAndProblem(_ context.Context, userID, problemID string) (*domainSubmission.Submission, error) {
	if m.lastFn != nil {
		return m.lastFn(userID, problemID)
	}
	return nil, apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "none")
}

func (m *mockSubmissionRepo) ExistsByHashAndUserAndProblem(_ context.Context, hash, userID, problemID string) (bool, error) {
	if m.existsDupFn != nil {
		return m.existsDupFn(hash, userID, problemID)
	}
	return false, nil
}

func cleanRepo() *mockSubmissionRepo { return &mockSubmissionRepo{} }

// ── mockSourceStorage ─────────────────────────────────────────────────────────

type mockSourceStorage struct {
	uploadErr error
	deleteErr error
}

func (m *mockSourceStorage) Upload(_ context.Context, _ string, _ []byte) error {
	return m.uploadErr
}

func (m *mockSourceStorage) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

// ── mockSubmissionQueue ───────────────────────────────────────────────────────

type mockQueue struct{}

func (m *mockQueue) Publish(_ context.Context, _ SubmissionQueueMessage) error { return nil }

// ── builder helper ────────────────────────────────────────────────────────────

func newUseCase(
	pp *mockProblemProvider,
	repo *mockSubmissionRepo,
	storage *mockSourceStorage,
) *SubmitSolutionUseCase {
	if storage == nil {
		storage = &mockSourceStorage{}
	}
	return NewSubmitSolutionUseCase(pp, repo, storage, &mockQueue{}, maxFileSize, rateLimitSeconds)
}

func validInput() SubmitSolutionInput {
	return SubmitSolutionInput{
		CurrentUser: callerUser(),
		ProblemSlug: testProblemSlug,
		Language:    "cpp20",
		Compiler:    "g++",
		FileName:    "solution.cpp",
		FileData:    []byte("#include <bits/stdc++.h>"),
		SubmittedAt: testNow,
	}
}

// restore a submission with given submittedAt — used for rate-limit tests
func recentSubmission(at time.Time) *domainSubmission.Submission {
	lang := domainSubmission.RestoreLanguage("cpp20")
	return domainSubmission.RestoreSubmission(
		"sub-001", testProblemID, shared.RestoreUserID(testUserID), nil, nil,
		lang, "g++", domainSubmission.RestoreStatus("PENDING"),
		"path/to/file.cpp", "abc123", 100, at, nil, nil, nil, nil,
	)
}
