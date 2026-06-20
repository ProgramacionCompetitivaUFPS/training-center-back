package submission

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mock dependencies ─────────────────────────────────────────────────────────

type mockProblemProvider struct {
	fn func(slug string) (*appSubmission.ProblemInfo, error)
}

func (m *mockProblemProvider) GetProblemBySlug(_ context.Context, slug string) (*appSubmission.ProblemInfo, error) {
	if m.fn != nil {
		return m.fn(slug)
	}
	return &appSubmission.ProblemInfo{
		ID:          "problem-001",
		Slug:        slug,
		Title:       "Test Problem",
		IsPublished: true,
		IsPublic:    true,
	}, nil
}

type mockSubmissionRepo struct {
	saveErr           error
	existsByHashFn    func(hash, userID, problemID string) (bool, error)
	findLastFn        func(userID, problemID string) (*domainSubmission.Submission, error)
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
	if m.findLastFn != nil {
		return m.findLastFn(userID, problemID)
	}
	return nil, apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "none")
}
func (m *mockSubmissionRepo) ExistsByHashAndUserAndProblem(_ context.Context, hash, userID, problemID string) (bool, error) {
	if m.existsByHashFn != nil {
		return m.existsByHashFn(hash, userID, problemID)
	}
	return false, nil
}

type mockStorage struct {
	uploadErr error
	deleteErr error
}

func (m *mockStorage) Upload(_ context.Context, _ string, _ []byte) error { return m.uploadErr }
func (m *mockStorage) Delete(_ context.Context, _ string) error           { return m.deleteErr }

type mockQueue struct{}

func (m *mockQueue) Publish(_ context.Context, _ appSubmission.SubmissionQueueMessage) error {
	return nil
}

// ── builder ───────────────────────────────────────────────────────────────────

func newHandlerWithSubmit(pp appSubmission.ProblemProvider, repo domainSubmission.Repository) *Handler {
	uc := appSubmission.NewSubmitSolutionUseCase(pp, repo, &mockStorage{}, &mockQueue{}, 1<<20, 1)
	return NewHandler(uc)
}

// ── auth helpers ──────────────────────────────────────────────────────────────

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{
		UserID: "user-001",
		Role:   domainShared.RoleContestant,
	}, nil
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

// ── multipart builder ─────────────────────────────────────────────────────────

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
