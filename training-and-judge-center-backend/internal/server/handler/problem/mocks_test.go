package problem

import (
	"context"
	"net/http"
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

// ── use case mocks ────────────────────────────────────────────────────────────

type mockCreateUC struct {
	fn func(context.Context, appProblem.CreateProblemInput) (*appProblem.CreateProblemResult, error)
}

func (m *mockCreateUC) Execute(ctx context.Context, in appProblem.CreateProblemInput) (*appProblem.CreateProblemResult, error) {
	if m.fn == nil {
		panic("mockCreateUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockUpdateUC struct {
	fn func(context.Context, appProblem.UpdateProblemInput) (*appProblem.UpdateProblemResult, error)
}

func (m *mockUpdateUC) Execute(ctx context.Context, in appProblem.UpdateProblemInput) (*appProblem.UpdateProblemResult, error) {
	if m.fn == nil {
		panic("mockUpdateUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockImportUC struct {
	fn func(context.Context, appProblem.ImportProblemInput) (*appProblem.ImportProblemResult, error)
}

func (m *mockImportUC) Execute(ctx context.Context, in appProblem.ImportProblemInput) (*appProblem.ImportProblemResult, error) {
	if m.fn == nil {
		panic("mockImportUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockDeleteProblemUC struct {
	fn func(context.Context, appProblem.DeleteProblemInput) (struct{}, error)
}

func (m *mockDeleteProblemUC) Execute(ctx context.Context, in appProblem.DeleteProblemInput) (struct{}, error) {
	if m.fn == nil {
		panic("mockDeleteProblemUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockGetProblemUC struct {
	fn func(context.Context, appProblem.GetProblemInput) (*appProblem.GetProblemOutput, error)
}

func (m *mockGetProblemUC) Execute(ctx context.Context, in appProblem.GetProblemInput) (*appProblem.GetProblemOutput, error) {
	if m.fn == nil {
		panic("mockGetProblemUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockListProblemsUC struct {
	fn func(context.Context, appProblem.ListProblemsInput) (*appProblem.ListProblemsOutput, error)
}

func (m *mockListProblemsUC) Execute(ctx context.Context, in appProblem.ListProblemsInput) (*appProblem.ListProblemsOutput, error) {
	if m.fn == nil {
		panic("mockListProblemsUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockUnpublishUC struct {
	fn func(context.Context, appProblem.UnpublishProblemInput) (*appProblem.UnpublishProblemOutput, error)
}

func (m *mockUnpublishUC) Execute(ctx context.Context, in appProblem.UnpublishProblemInput) (*appProblem.UnpublishProblemOutput, error) {
	if m.fn == nil {
		panic("mockUnpublishUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

type mockChangeAccessibilityUC struct {
	fn func(context.Context, appProblem.ChangeAccessibilityInput) (*appProblem.ChangeAccessibilityOutput, error)
}

func (m *mockChangeAccessibilityUC) Execute(ctx context.Context, in appProblem.ChangeAccessibilityInput) (*appProblem.ChangeAccessibilityOutput, error) {
	if m.fn == nil {
		panic("mockChangeAccessibilityUC.Execute called unexpectedly")
	}
	return m.fn(ctx, in)
}

// noopUC satisfies interfaces that a test doesn't care about — panics if called.
type noopUploadUC struct{}

func (n *noopUploadUC) Execute(ctx context.Context, in appProblem.UploadProblemFilesInput) (*appProblem.UploadProblemFilesResult, error) {
	panic("noopUploadUC.Execute called unexpectedly")
}

type noopDeleteFileUC struct{}

func (n *noopDeleteFileUC) Execute(ctx context.Context, in appProblem.DeleteProblemFileInput) (struct{}, error) {
	panic("noopDeleteFileUC.Execute called unexpectedly")
}

type noopAddModifierUC struct{}

func (n *noopAddModifierUC) Execute(ctx context.Context, in appProblem.AddModifierInput) (struct{}, error) {
	panic("noopAddModifierUC.Execute called unexpectedly")
}

type noopRemoveModifierUC struct{}

func (n *noopRemoveModifierUC) Execute(ctx context.Context, in appProblem.RemoveModifierInput) (struct{}, error) {
	panic("noopRemoveModifierUC.Execute called unexpectedly")
}

type noopListModifiersUC struct{}

func (n *noopListModifiersUC) Execute(ctx context.Context, slugStr string, cu user.CurrentUser) ([]string, error) {
	panic("noopListModifiersUC.Execute called unexpectedly")
}

// ── user provider mock ────────────────────────────────────────────────────────

type mockUserProvider struct {
	displayFn func(context.Context, string) (*user.Display, error)
}

func (m *mockUserProvider) GetDisplay(ctx context.Context, id string) (*user.Display, error) {
	if m.displayFn != nil {
		return m.displayFn(ctx, id)
	}
	return &user.Display{Nickname: "author", Name: "Author Name"}, nil
}

func (m *mockUserProvider) GetDisplays(ctx context.Context, ids []string) (map[string]*user.Display, error) {
	result := make(map[string]*user.Display)
	for _, id := range ids {
		result[id] = &user.Display{Nickname: "author", Name: "Author Name"}
	}
	return result, nil
}

func (m *mockUserProvider) ExistsByID(ctx context.Context, id string) (bool, error) {
	return true, nil
}

func (m *mockUserProvider) GetIDByNickname(ctx context.Context, nickname string) (string, bool, error) {
	return "some-id", true, nil
}

// ── settings mock ─────────────────────────────────────────────────────────────

type mockSettings struct{}

func (m *mockSettings) IsLanguageSupported(language string) bool        { return true }
func (m *mockSettings) GetLanguageByExtension(ext string) (string, bool) { return "cpp20", true }
func (m *mockSettings) GetUploadMaxConcurrency() int                    { return 1 }
func (m *mockSettings) GetMaxFileCountSample() int                      { return 10 }
func (m *mockSettings) GetMaxFileSizeDefaultMB() int                    { return 10 }
func (m *mockSettings) GetMaxFileSizeTestCaseMB() int                   { return 50 }
func (m *mockSettings) GetMaxFileCountTestCase() int                    { return 100 }
func (m *mockSettings) GetGlobalLimits() (int, int)                     { return 5000, 512 }
func (m *mockSettings) GetLanguageLimit(language string) *domainProblem.LanguageLimit { return nil }
func (m *mockSettings) IsValidTag(tag string) bool                      { return true }
func (m *mockSettings) GetAllowedTags() map[string]struct{}             { return map[string]struct{}{} }

// ── test helpers ──────────────────────────────────────────────────────────────

var (
	coachUser = user.CurrentUser{ID: "coach-id", Role: user.RoleCoach}
	adminUser = user.CurrentUser{ID: "admin-id", Role: user.RoleAdmin}
)

func withUser(r *http.Request, u user.CurrentUser) *http.Request {
	return r.WithContext(middleware.WithCurrentUser(r.Context(), u))
}

func testProblem(slug string) *domainProblem.Problem {
	now := time.Now()
	return domainProblem.RestoreProblem(
		"test-id", slug, "Test Problem",
		nil, nil, nil,
		[]string{},
		"DRAFT", "PRIVATE",
		domainProblem.RestoreUserID("author-id"),
		nil, nil, nil, nil, nil, nil, nil,
		now, now,
	)
}

func newTestHandler(
	createUC createProblemUC,
	updateUC updateProblemUC,
	importUC importProblemUC,
	getProblemUC getProblemUC,
	listProblemsUC listProblemsUC,
	deleteProblemUC deleteProblemUC,
	unpublishUC unpublishProblemUC,
	changeAccessibilityUC changeAccessibilityProblemUC,
) *Handler {
	return NewHandler(
		createUC,
		importUC,
		updateUC,
		&noopUploadUC{},
		&noopDeleteFileUC{},
		&noopAddModifierUC{},
		&noopRemoveModifierUC{},
		&noopListModifiersUC{},
		getProblemUC,
		listProblemsUC,
		unpublishUC,
		changeAccessibilityUC,
		deleteProblemUC,
		&mockUserProvider{},
		&mockSettings{},
	)
}
