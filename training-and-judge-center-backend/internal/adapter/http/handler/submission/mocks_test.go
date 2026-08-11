package submission

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/auth"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

// ── mock dependencies ─────────────────────────────────────────────────────────

type mockProblemProvider struct {
	fn func(slug string) (*appsubmission.ProblemInfo, error)
}

func (m *mockProblemProvider) GetProblemBySlug(_ context.Context, slug string) (*appsubmission.ProblemInfo, error) {
	if m.fn != nil {
		return m.fn(slug)
	}
	return &appsubmission.ProblemInfo{
		ID:          "problem-001",
		Slug:        slug,
		Title:       "Test Problem",
		IsPublished: true,
		IsPublic:    true,
	}, nil
}

type mockSubmissionRepo struct {
	saveErr        error
	existsByHashFn func(hash, userID, problemID string) (bool, error)
	findLastFn     func(userID, problemID string) (*domainsubmission.Submission, error)
	findByIDFn     func(id string) (*domainsubmission.Submission, error)
	updateVisFn    func(s *domainsubmission.Submission) error
	listFn         func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error)
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
	if m.findLastFn != nil {
		return m.findLastFn(userID, problemID)
	}
	return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "none")
}
func (m *mockSubmissionRepo) ExistsByHashAndUserAndProblem(_ context.Context, hash, userID, problemID string, _ *string) (bool, error) {
	if m.existsByHashFn != nil {
		return m.existsByHashFn(hash, userID, problemID)
	}
	return false, nil
}
func (m *mockSubmissionRepo) UpdateVisibility(_ context.Context, s *domainsubmission.Submission) error {
	if m.updateVisFn != nil {
		return m.updateVisFn(s)
	}
	return nil
}

type mockStorage struct {
	uploadErr error
	deleteErr error
}

func (m *mockStorage) Upload(_ context.Context, _ string, _ []byte) error { return m.uploadErr }
func (m *mockStorage) Delete(_ context.Context, _ string) error           { return m.deleteErr }

type mockQueue struct{}

func (m *mockQueue) Publish(_ context.Context, _ appsubmission.SubmissionQueueMessage) error {
	return nil
}

// ── builder ───────────────────────────────────────────────────────────────────

func newHandlerWithSubmit(pp appsubmission.ProblemProvider, repo domainsubmission.Repository) *Handler {
	uc := appsubmission.NewSubmitSolutionUseCase(pp, repo, &mockStorage{}, &mockQueue{}, 1<<20, 1)
	return NewHandler(uc, nil, nil, nil, nil, nil, nil)
}

// ── auth helpers ──────────────────────────────────────────────────────────────

type mockTokenSvc struct {
	validateFn func(token string) (*domainuser.TokenClaims, error)
}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainuser.User, _ string) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(token string) (*domainuser.TokenClaims, error) {
	if m.validateFn != nil {
		return m.validateFn(token)
	}
	return &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}, nil
}

func wrapWithAuth(h http.Handler, claims *domainuser.TokenClaims) http.Handler {
	tokenSvc := &mockTokenSvc{
		validateFn: func(_ string) (*domainuser.TokenClaims, error) {
			return claims, nil
		},
	}
	return middleware.Auth(tokenSvc, &auth.NoOpSessionInvalidator{})(h)
}

// ── mocks for get/update-visibility use cases ─────────────────────────────────

type mockSourceCodeReader struct {
	fn func(path string) ([]byte, error)
}

func (m *mockSourceCodeReader) Read(_ context.Context, path string) ([]byte, error) {
	if m.fn != nil {
		return m.fn(path)
	}
	return []byte("int main(){}"), nil
}

type mockProblemDisplayProvider struct {
	fn func(id string) (*appsubmission.ProblemDisplay, error)
}

func (m *mockProblemDisplayProvider) GetProblemByID(_ context.Context, id string) (*appsubmission.ProblemDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &appsubmission.ProblemDisplay{Slug: "two-sum", Title: "Two Sum"}, nil
}

type mockUserDisplayProvider struct {
	fn func(id string) (*appsubmission.UserDisplay, error)
}

func (m *mockUserDisplayProvider) GetUserByID(_ context.Context, id string) (*appsubmission.UserDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &appsubmission.UserDisplay{ID: id, Nickname: "author"}, nil
}

type mockContestDisplayProvider struct {
	fn func(id string) (*appsubmission.ContestDisplay, error)
}

func (m *mockContestDisplayProvider) GetContestByID(_ context.Context, id string) (*appsubmission.ContestDisplay, error) {
	if m.fn != nil {
		return m.fn(id)
	}
	return &appsubmission.ContestDisplay{ID: id, Name: "Test Contest"}, nil
}

type mockTeamChecker struct {
	fn func(contestID, viewerID, authorID string) (bool, error)
}

func (m *mockTeamChecker) IsTeammate(_ context.Context, contestID, viewerID, authorID string) (bool, error) {
	if m.fn != nil {
		return m.fn(contestID, viewerID, authorID)
	}
	return false, nil
}

type mockLeadChecker struct {
	fn func(contestID, userID string) (bool, error)
}

func (m *mockLeadChecker) IsLeadOfContestGroup(_ context.Context, contestID, userID string) (bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userID)
	}
	return false, nil
}

// ── mocks for contest submission ──────────────────────────────────────────────

type mockContestSubmissionProvider struct {
	fn func(groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error)
}

func (m *mockContestSubmissionProvider) GetContestForSubmission(_ context.Context, groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
	if m.fn != nil {
		return m.fn(groupID, contestID)
	}
	// Default: ACTIVE contest with "problem-001"
	return &appsubmission.ContestSubmissionInfo{
		ID:                contestID,
		Name:              "Weekly Contest #1",
		GroupID:           groupID,
		StartTime:         time.Now().Add(-1 * time.Hour),
		EndTime:           time.Now().Add(1 * time.Hour),
		EnablePostContest: false,
		ProblemIDs:        []string{"problem-001"},
	}, nil
}

type mockStandingIDResolver struct {
	fn func(contestID, userID string) (string, bool, error)
}

func (m *mockStandingIDResolver) ResolveStandingID(_ context.Context, contestID, userID string) (string, bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userID)
	}
	return userID, true, nil // default: registered individually
}

func newHandlerWithSubmitContest(
	contestProvider appsubmission.ContestSubmissionProvider,
	standingResolver appsubmission.StandingIDResolver,
	problemProvider appsubmission.ProblemProvider,
	repo domainsubmission.Repository,
) *Handler {
	uc := appsubmission.NewSubmitContestSolutionUseCase(
		contestProvider, standingResolver, problemProvider, repo,
		&mockStorage{}, &mockQueue{}, 1<<20, 1,
	)
	return NewHandler(nil, uc, nil, nil, nil, nil, nil)
}

// ── fixtures ──────────────────────────────────────────────────────────────────

const (
	testAuthorID     = "aaaaaaaa-0000-0000-0000-000000000001"
	testSubmissionID = "bbbbbbbb-0000-0000-0000-000000000001"
	testProblemID    = "cccccccc-0000-0000-0000-000000000001"
)

func aSubmission(authorID, visibility string, contestID *string) *domainsubmission.Submission {
	lang := domainsubmission.RestoreLanguage("cpp20")
	return domainsubmission.RestoreSubmission(
		testSubmissionID, testProblemID, domainshared.RestoreUserID(authorID),
		contestID, nil,
		lang, "g++",
		domainsubmission.RestoreStatus("PENDING"),
		domainsubmission.RestoreVisibility(visibility),
		"gs://bucket/code.cpp", "hash", 100,
		testNow, nil, nil, nil, nil, "", "",
	)
}

// ── builders ──────────────────────────────────────────────────────────────────

func newHandlerWithGetSubmission(repo *mockSubmissionRepo) *Handler {
	uc := appsubmission.NewGetSubmissionUseCase(
		repo,
		&mockSourceCodeReader{},
		&mockProblemDisplayProvider{},
		&mockUserDisplayProvider{},
		&mockContestDisplayProvider{},
		&mockTeamChecker{},
		&mockLeadChecker{},
	)
	return NewHandler(nil, nil, uc, nil, nil, nil, nil)
}

func newHandlerWithUpdateVisibility(repo *mockSubmissionRepo) *Handler {
	uc := appsubmission.NewUpdateSubmissionVisibilityUseCase(repo)
	return NewHandler(nil, nil, nil, uc, nil, nil, nil)
}

func newHandlerWithListMy(repo *mockSubmissionRepo) *Handler {
	uc := appsubmission.NewListMySubmissionsUseCase(
		repo,
		&mockProblemDisplayProvider{},
		&mockUserDisplayProvider{},
		&mockContestDisplayProvider{},
		&mockProblemProvider{},
	)
	return NewHandler(nil, nil, nil, nil, uc, nil, nil)
}

func newHandlerWithListProblem(repo *mockSubmissionRepo) *Handler {
	uc := appsubmission.NewListProblemSubmissionsUseCase(
		repo,
		&mockProblemProvider{},
		&mockUserDisplayProvider{},
	)
	return NewHandler(nil, nil, nil, nil, nil, uc, nil)
}

// ── multipart builders ────────────────────────────────────────────────────────

func multipartRequest(target, slug, language, compiler, filename string, fileData []byte) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	_ = w.WriteField("language", language)
	_ = w.WriteField("compiler", compiler)

	part, _ := w.CreateFormFile("file", filename)
	_, _ = part.Write(fileData)
	w.Close()

	r := httptest.NewRequest(http.MethodPost, target, body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.SetPathValue("slug", slug)
	return r
}

func contestMultipartRequest(groupID, contestID, problemSlug, language, compiler, filename string, fileData []byte) *http.Request {
	target := fmt.Sprintf("/groups/%s/contests/%s/problems/%s/submissions", groupID, contestID, problemSlug)
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	_ = w.WriteField("language", language)
	_ = w.WriteField("compiler", compiler)

	part, _ := w.CreateFormFile("file", filename)
	_, _ = part.Write(fileData)
	w.Close()

	r := httptest.NewRequest(http.MethodPost, target, body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.SetPathValue("groupId", groupID)
	r.SetPathValue("contestId", contestID)
	r.SetPathValue("problemSlug", problemSlug)
	return r
}
