package submission

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
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
	lastFn       func(userID, problemID string) (*domainsubmission.Submission, error)
	findByIDFn   func(id string) (*domainsubmission.Submission, error)
	updateVisFn  func(s *domainsubmission.Submission) error
	listFn       func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error)
}

func (m *mockSubmissionRepo) Save(_ context.Context, _ *domainsubmission.Submission) error {
	return m.saveErr
}

func (m *mockSubmissionRepo) FindByID(_ context.Context, id string) (*domainsubmission.Submission, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "not found")
}

func (m *mockSubmissionRepo) List(_ context.Context, f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
	if m.listFn != nil {
		return m.listFn(f)
	}
	return []*domainsubmission.Submission{}, 0, nil
}

func (m *mockSubmissionRepo) FindLastByUserAndProblem(_ context.Context, userID, problemID string, _ *string) (*domainsubmission.Submission, error) {
	if m.lastFn != nil {
		return m.lastFn(userID, problemID)
	}
	return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "none")
}

func (m *mockSubmissionRepo) ExistsByHashAndUserAndProblem(_ context.Context, hash, userID, problemID string, _ *string) (bool, error) {
	if m.existsDupFn != nil {
		return m.existsDupFn(hash, userID, problemID)
	}
	return false, nil
}

func (m *mockSubmissionRepo) UpdateVisibility(_ context.Context, s *domainsubmission.Submission) error {
	if m.updateVisFn != nil {
		return m.updateVisFn(s)
	}
	return nil
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

type mockQueue struct {
	published []SubmissionQueueMessage
	err       error
}

func (m *mockQueue) Publish(_ context.Context, msg SubmissionQueueMessage) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, msg)
	return nil
}

func (m *mockQueue) lastPublished() *SubmissionQueueMessage {
	if len(m.published) == 0 {
		return nil
	}
	return &m.published[len(m.published)-1]
}

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
func recentSubmission(at time.Time) *domainsubmission.Submission {
	lang := domainsubmission.RestoreLanguage("cpp20")
	return domainsubmission.RestoreSubmission(
		"sub-001", testProblemID, shared.RestoreUserID(testUserID), nil, nil,
		lang, "g++", domainsubmission.RestoreStatus("PENDING"),
		domainsubmission.RestoreVisibility("PRIVATE"),
		"path/to/file.cpp", "abc123", 100, at, nil, nil, nil, nil, "", "",
	)
}

// ── fixtures for get/update visibility tests ──────────────────────────────────

const (
	testSubmissionID = "bbbbbbbb-0000-0000-0000-000000000001"
	testAuthorID     = testUserID
	testOtherUserID  = "ffffffff-0000-0000-0000-000000000001"
	testContestID    = "cccccccc-0000-0000-0000-000000000001"
	testGroupID      = "eeeeeeee-0000-0000-0000-000000000001"
	testSourcePath   = "gs://bucket/code.cpp"
)

func aSubmission(authorID, visibility string, contestID *string) *domainsubmission.Submission {
	lang := domainsubmission.RestoreLanguage("cpp20")
	return domainsubmission.RestoreSubmission(
		testSubmissionID, testProblemID, shared.RestoreUserID(authorID),
		contestID, nil,
		lang, "g++",
		domainsubmission.RestoreStatus("PENDING"),
		domainsubmission.RestoreVisibility(visibility),
		testSourcePath, "hash", 100,
		testNow, nil, nil, nil, nil, "", "",
	)
}

// ── mockSourceCodeReader ──────────────────────────────────────────────────────

type mockSourceCodeReader struct {
	fn func(path string) ([]byte, error)
}

func (m *mockSourceCodeReader) Read(_ context.Context, path string) ([]byte, error) {
	if m.fn != nil {
		return m.fn(path)
	}
	return []byte("int main(){}"), nil
}

// ── mockProblemDisplayProvider ────────────────────────────────────────────────

type mockProblemDisplayProvider struct {
	fn func(problemID string) (*ProblemDisplay, error)
}

func (m *mockProblemDisplayProvider) GetProblemByID(_ context.Context, id string) (*ProblemDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &ProblemDisplay{Slug: "two-sum", Title: "Two Sum"}, nil
}

// ── mockUserProvider ──────────────────────────────────────────────────────────

type mockUserProvider struct {
	fn func(userID string) (*UserDisplay, error)
}

func (m *mockUserProvider) GetUserByID(_ context.Context, id string) (*UserDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &UserDisplay{ID: id, Nickname: "testuser"}, nil
}

// ── mockContestProvider ───────────────────────────────────────────────────────

type mockContestProvider struct {
	fn func(contestID string) (*ContestDisplay, error)
}

func (m *mockContestProvider) GetContestByID(_ context.Context, id string) (*ContestDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &ContestDisplay{ID: id, Name: "Test Contest"}, nil
}

// ── mockTeamMembershipChecker ─────────────────────────────────────────────────

type mockTeamMembershipChecker struct {
	fn func(contestID, viewerID, authorID string) (bool, error)
}

func (m *mockTeamMembershipChecker) IsTeammate(_ context.Context, contestID, viewerID, authorID string) (bool, error) {
	if m.fn != nil {
		return m.fn(contestID, viewerID, authorID)
	}
	return false, nil
}

// ── mockLeadChecker ───────────────────────────────────────────────────────────

type mockLeadChecker struct {
	fn func(contestID, userID string) (bool, error)
}

func (m *mockLeadChecker) IsLeadOfContestGroup(_ context.Context, contestID, userID string) (bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userID)
	}
	return false, nil
}

// ── builders ──────────────────────────────────────────────────────────────────

func newGetUseCase(
	repo *mockSubmissionRepo,
	reader *mockSourceCodeReader,
	prob *mockProblemDisplayProvider,
	usr *mockUserProvider,
	cont *mockContestProvider,
	team *mockTeamMembershipChecker,
	lead *mockLeadChecker,
) *GetSubmissionUseCase {
	if reader == nil {
		reader = &mockSourceCodeReader{}
	}
	if prob == nil {
		prob = &mockProblemDisplayProvider{}
	}
	if usr == nil {
		usr = &mockUserProvider{}
	}
	if cont == nil {
		cont = &mockContestProvider{}
	}
	if team == nil {
		team = &mockTeamMembershipChecker{}
	}
	if lead == nil {
		lead = &mockLeadChecker{}
	}
	return NewGetSubmissionUseCase(repo, reader, prob, usr, cont, team, lead)
}

func newUpdateVisUseCase(repo *mockSubmissionRepo) *UpdateSubmissionVisibilityUseCase {
	return NewUpdateSubmissionVisibilityUseCase(repo)
}

func newListMyUseCase(
	repo *mockSubmissionRepo,
	prob *mockProblemDisplayProvider,
	usr *mockUserProvider,
	cont *mockContestProvider,
	bySlug *mockProblemProvider,
) *ListMySubmissionsUseCase {
	if prob == nil {
		prob = &mockProblemDisplayProvider{}
	}
	if usr == nil {
		usr = &mockUserProvider{}
	}
	if cont == nil {
		cont = &mockContestProvider{}
	}
	if bySlug == nil {
		bySlug = &mockProblemProvider{}
	}
	return NewListMySubmissionsUseCase(repo, prob, usr, cont, bySlug)
}

func newListProblemSubmissionsUseCase(
	repo *mockSubmissionRepo,
	bySlug *mockProblemProvider,
	usr *mockUserProvider,
) *ListProblemSubmissionsUseCase {
	if bySlug == nil {
		bySlug = &mockProblemProvider{}
	}
	if usr == nil {
		usr = &mockUserProvider{}
	}
	return NewListProblemSubmissionsUseCase(repo, bySlug, usr)
}

// ── mockContestSubmissionProvider ─────────────────────────────────────────────

type mockContestSubmissionProvider struct {
	fn func(groupID, contestID string) (*ContestSubmissionInfo, error)
}

func (m *mockContestSubmissionProvider) GetContestForSubmission(_ context.Context, groupID, contestID string) (*ContestSubmissionInfo, error) {
	if m.fn != nil {
		return m.fn(groupID, contestID)
	}
	return nil, apperror.NewNotFound("CONTEST_NOT_FOUND", "contest not found")
}

// ── mockStandingIDResolver ────────────────────────────────────────────────────

type mockStandingIDResolver struct {
	fn func(contestID, userID string) (string, bool, error)
}

func (m *mockStandingIDResolver) ResolveStandingID(_ context.Context, contestID, userID string) (string, bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userID)
	}
	return "standing-001", true, nil
}
