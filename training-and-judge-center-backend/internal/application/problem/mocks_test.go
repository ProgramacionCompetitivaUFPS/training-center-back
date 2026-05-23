package problem

import (
	"context"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
)

var testNow = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

// ── Repository mock ──────────────────────────────────────────────────────────

type mockProblemRepository struct {
	saveFn         func(ctx context.Context, p *domainProblem.Problem) error
	findBySlugFn   func(ctx context.Context, slug domainProblem.Slug) (*domainProblem.Problem, error)
	existsBySlugFn func(ctx context.Context, slug domainProblem.Slug) (bool, error)
	listFn         func(ctx context.Context, f domainProblem.ListFilters) ([]*domainProblem.Problem, int, error)
	deleteFn       func(ctx context.Context, id string) error
}

func (m *mockProblemRepository) Save(ctx context.Context, p *domainProblem.Problem) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, p)
	}
	return nil
}
func (m *mockProblemRepository) FindBySlug(ctx context.Context, slug domainProblem.Slug) (*domainProblem.Problem, error) {
	if m.findBySlugFn != nil {
		return m.findBySlugFn(ctx, slug)
	}
	return nil, nil
}
func (m *mockProblemRepository) ExistsBySlug(ctx context.Context, slug domainProblem.Slug) (bool, error) {
	if m.existsBySlugFn != nil {
		return m.existsBySlugFn(ctx, slug)
	}
	return false, nil
}
func (m *mockProblemRepository) List(ctx context.Context, f domainProblem.ListFilters) ([]*domainProblem.Problem, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockProblemRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ── UserProvider mock ────────────────────────────────────────────────────────

type mockUserProvider struct {
	existsByIDFn      func(ctx context.Context, userID string) (bool, error)
	getDisplayFn      func(ctx context.Context, userID string) (*UserDisplay, error)
	getDisplaysFn     func(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
	getIDByNicknameFn func(ctx context.Context, nickname string) (string, bool, error)
}

func (m *mockUserProvider) ExistsByID(ctx context.Context, userID string) (bool, error) {
	if m.existsByIDFn != nil {
		return m.existsByIDFn(ctx, userID)
	}
	return true, nil
}
func (m *mockUserProvider) GetDisplay(ctx context.Context, userID string) (*UserDisplay, error) {
	if m.getDisplayFn != nil {
		return m.getDisplayFn(ctx, userID)
	}
	return &UserDisplay{Nickname: "user_" + userID[:4], Name: "Test User"}, nil
}
func (m *mockUserProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error) {
	if m.getDisplaysFn != nil {
		return m.getDisplaysFn(ctx, userIDs)
	}
	out := make(map[string]*UserDisplay, len(userIDs))
	for _, id := range userIDs {
		out[id] = &UserDisplay{Nickname: "user_" + id[:4], Name: "Test User"}
	}
	return out, nil
}
func (m *mockUserProvider) GetIDByNickname(ctx context.Context, nickname string) (string, bool, error) {
	if m.getIDByNicknameFn != nil {
		return m.getIDByNicknameFn(ctx, nickname)
	}
	return "", false, nil
}

// ── ProblemFileRepository mock ───────────────────────────────────────────────

type mockFileStorage struct {
	uploadFileFn            func(ctx context.Context, path string, content []byte) error
	deleteFileFn            func(ctx context.Context, path string) error
	deleteFilesWithPrefixFn func(ctx context.Context, prefix string) error
}

func (m *mockFileStorage) UploadFile(ctx context.Context, path string, content []byte) error {
	if m.uploadFileFn != nil {
		return m.uploadFileFn(ctx, path, content)
	}
	return nil
}
func (m *mockFileStorage) DeleteFile(ctx context.Context, path string) error {
	if m.deleteFileFn != nil {
		return m.deleteFileFn(ctx, path)
	}
	return nil
}
func (m *mockFileStorage) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	if m.deleteFilesWithPrefixFn != nil {
		return m.deleteFilesWithPrefixFn(ctx, prefix)
	}
	return nil
}

// ── PlatformSettings default ─────────────────────────────────────────────────

func newDefaultSettings() domainProblem.PlatformSettings {
	s, err := domainProblem.NewPlatformSettings(
		10000,
		512,
		map[string]domainProblem.LanguageLimit{},
		map[string]struct{}{"cpp": {}},
		map[string]string{".cpp": "cpp"},
		map[string]struct{}{"dp": {}, "graphs": {}, "implementation": {}},
		4,
		10,
		50,
		100,
		10,
	)
	if err != nil {
		panic(err)
	}
	return s
}

// ── CurrentUser helpers ──────────────────────────────────────────────────────

var (
	asAdmin      = testutil.AsAdmin
	asCoach      = testutil.AsCoach
	asContestant = testutil.AsContestant
)

// ── Problem fixtures ─────────────────────────────────────────────────────────

const (
	authorID    = "aaaaaaaa-0000-0000-0000-000000000001"
	modifierID  = "aaaaaaaa-0000-0000-0000-000000000002"
	strangerID  = "aaaaaaaa-0000-0000-0000-000000000003"
	testSlug    = "test-problem"
	testProbID  = "bbbbbbbb-0000-0000-0000-000000000001"
)

func newDraftProblem() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"DRAFT", "PRIVATE",
		shared.RestoreUserID(authorID),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func newPublishedProblem() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(authorID),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func newDraftProblemWithModifier() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"DRAFT", "PRIVATE",
		shared.RestoreUserID(authorID),
		[]shared.UserID{shared.RestoreUserID(modifierID)},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func newPublishedProblemWithModifier() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		testProbID, testSlug, "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(authorID),
		[]shared.UserID{shared.RestoreUserID(modifierID)},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func repoWith(p *domainProblem.Problem) *mockProblemRepository {
	return &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return p, nil
		},
	}
}
