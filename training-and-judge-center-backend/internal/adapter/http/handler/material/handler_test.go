package material

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Repository stub ───────────────────────────────────────────────────────────

type stubMaterialRepo struct {
	findByIDFn func(ctx context.Context, id string) (*domainMaterial.Material, error)
}

func (s *stubMaterialRepo) Save(_ context.Context, _ *domainMaterial.Material) error { return nil }
func (s *stubMaterialRepo) FindByID(ctx context.Context, id string) (*domainMaterial.Material, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "not found")
}
func (s *stubMaterialRepo) List(_ context.Context, _ string, _ domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
	return nil, 0, nil
}
func (s *stubMaterialRepo) Delete(_ context.Context, _ string) error { return nil }

// ── Provider stubs ────────────────────────────────────────────────────────────

type stubGroupProvider struct{}

func (s *stubGroupProvider) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

type stubGroupVisibilityProvider struct{}

func (s *stubGroupVisibilityProvider) FindVisibility(_ context.Context, _ string) (appMaterial.GroupVisibility, bool, error) {
	return appMaterial.GroupVisibilityVisible, true, nil
}

type stubNotVisibleGroupVisibilityProvider struct{}

func (s *stubNotVisibleGroupVisibilityProvider) FindVisibility(_ context.Context, _ string) (appMaterial.GroupVisibility, bool, error) {
	return appMaterial.GroupVisibilityNotVisible, true, nil
}

type stubGroupMemberProvider struct{}

func (s *stubGroupMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (s *stubGroupMemberProvider) IsMemberOfGroup(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

type stubNotLeadMemberProvider struct{}

func (s *stubNotLeadMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubNotLeadMemberProvider) IsMemberOfGroup(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

type stubNonMemberProvider struct{}

func (s *stubNonMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *stubNonMemberProvider) IsMemberOfGroup(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

type stubAuthorProvider struct{}

func (s *stubAuthorProvider) GetDisplays(_ context.Context, ids []string) (map[string]*appMaterial.AuthorDisplay, error) {
	out := make(map[string]*appMaterial.AuthorDisplay, len(ids))
	for _, id := range ids {
		out[id] = &appMaterial.AuthorDisplay{Nickname: "author", Name: "Author Name"}
	}
	return out, nil
}

type stubAuthorIDProvider struct{}

func (s *stubAuthorIDProvider) FindIDByNickname(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

// ── Auth helpers ──────────────────────────────────────────────────────────────

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User, _ string) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: shared.RestoreUserID("u1").Value(), Role: shared.RoleCoach}, nil
}

func authedRequest(method, target string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	r.Header.Set("Authorization", "Bearer tok")
	return r
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, noopSessionInvalidator{})(h)
}

// noopSessionInvalidator is a package-local no-op user.SessionInvalidator so
// handler tests do not depend on the sibling adapter/auth package.
type noopSessionInvalidator struct{}

func (noopSessionInvalidator) InvalidateAllUserSessions(context.Context, string, time.Time) error {
	return nil
}
func (noopSessionInvalidator) IsAllUserSessionRevoked(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (noopSessionInvalidator) InvalidateSession(context.Context, string, time.Time) error {
	return nil
}
func (noopSessionInvalidator) IsSessionInvalidated(context.Context, string) (bool, error) {
	return false, nil
}
