package material

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Stubs ────────────────────────────────────────────────────────────────────

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

type stubGroupProvider struct{}

func (s *stubGroupProvider) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

type stubGroupVisibilityProvider struct{}

func (s *stubGroupVisibilityProvider) FindVisibility(_ context.Context, _ string) (appMaterial.GroupVisibility, bool, error) {
	return appMaterial.GroupVisibilityVisible, true, nil
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

type stubNotVisibleGroupVisibilityProvider struct{}

func (s *stubNotVisibleGroupVisibilityProvider) FindVisibility(_ context.Context, _ string) (appMaterial.GroupVisibility, bool, error) {
	return appMaterial.GroupVisibilityNotVisible, true, nil
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

func stubHandler() *Handler {
	return handlerWithRepo(&stubMaterialRepo{})
}

func handlerWithRepo(repo domainMaterial.Repository) *Handler {
	return NewHandler(
		appMaterial.NewCreateMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUpdateMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewGetMaterial(repo, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewListMaterials(repo, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPublishMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpublishMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPinMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpinMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewDeleteMaterial(repo, &stubGroupProvider{}),
	)
}

// ── Auth helpers ─────────────────────────────────────────────────────────────

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ *domainUser.User) (string, error) { return "tok", nil }
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: shared.RestoreUserID("u1").Value(), Role: domainUser.RoleCoach}, nil
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
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

// ── Create handler tests ──────────────────────────────────────────────────────

func TestCreateMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Create).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateMaterial_EmptyGroupId_Returns400(t *testing.T) {
	h := stubHandler()
	body, _ := json.Marshal(map[string]string{"title": "Hello"})
	r := authedRequest(http.MethodPost, "/groups//materials", body)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != apperror.ErrCodeBadRequest {
		t.Errorf("expected BAD_REQUEST, got %s", resp.Code)
	}
}

func TestCreateMaterial_InvalidBody_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPost, "/groups/g1/materials", []byte("not-json"))
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestCreateMaterial_ValidRequest_Returns201(t *testing.T) {
	h := stubHandler()
	body, _ := json.Marshal(map[string]string{"title": "My Material"})
	r := authedRequest(http.MethodPost, "/groups/g1/materials", body)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Title != "My Material" {
		t.Errorf("expected title 'My Material', got %q", resp.Title)
	}
	if resp.Status != "DRAFT" {
		t.Errorf("expected status DRAFT, got %q", resp.Status)
	}
}

func TestCreateMaterial_Forbidden_Returns403(t *testing.T) {
	h := NewHandler(
		appMaterial.NewCreateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubNotLeadMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUpdateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewGetMaterial(&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewListMaterials(&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewDeleteMaterial(&stubMaterialRepo{}, &stubGroupProvider{}),
	)

	body, _ := json.Marshal(map[string]string{"title": "My Material"})
	r := authedRequest(http.MethodPost, "/groups/g1/materials", body)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != appMaterial.ErrCodeInsufficientPerms {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %s", resp.Code)
	}
}

func TestUpdateMaterial_ValidRequest_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Old Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	repo := &stubMaterialRepo{}
	h := NewHandler(
		appMaterial.NewCreateMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUpdateMaterial(&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewGetMaterial(&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewListMaterials(&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewDeleteMaterial(&stubMaterialRepo{}, &stubGroupProvider{}),
	)

	body, _ := json.Marshal(map[string]string{"title": "Updated"})
	r := authedRequest(http.MethodPatch, "/groups/g1/materials/m1", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", resp.Title)
	}
}

func TestUpdateMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	body, _ := json.Marshal(map[string]string{"title": "X"})
	r := authedRequest(http.MethodPatch, "/groups/g1/materials/missing", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPatch, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Update).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateMaterial_MissingPathParams_Returns400(t *testing.T) {
	h := stubHandler()
	body, _ := json.Marshal(map[string]string{"title": "New"})
	r := authedRequest(http.MethodPatch, "/groups//materials/", body)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != apperror.ErrCodeBadRequest {
		t.Errorf("expected BAD_REQUEST, got %s", resp.Code)
	}
}

func TestUpdateMaterial_InvalidBody_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPatch, "/groups/g1/materials/m1", []byte("not-json"))
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

// ── Get handler tests ─────────────────────────────────────────────────────────

func TestGetMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Get).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetMaterial_MissingPathParams_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups//materials/", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler() // stubMaterialRepo returns 404 by default
	r := authedRequest(http.MethodGet, "/groups/g1/materials/missing", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetMaterial_ValidRequest_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", false, nil,
		time.Now(), time.Now(), nil,
	)
	repo := &stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	}
	h := NewHandler(
		appMaterial.NewCreateMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUpdateMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewGetMaterial(repo, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewListMaterials(repo, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPublishMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpublishMaterial(repo, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPinMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpinMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewDeleteMaterial(repo, &stubGroupProvider{}),
	)

	r := authedRequest(http.MethodGet, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Title != "Title" {
		t.Errorf("expected title 'Title', got %q", resp.Title)
	}
	if resp.Author == nil {
		t.Error("expected author to be populated")
	}
}

func TestGetMaterial_Forbidden_Returns403(t *testing.T) {
	h := NewHandler(
		appMaterial.NewCreateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUpdateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewGetMaterial(&stubMaterialRepo{}, &stubNotVisibleGroupVisibilityProvider{}, &stubNonMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewListMaterials(&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpublishMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{}),
		appMaterial.NewPinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewUnpinMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{}),
		appMaterial.NewDeleteMaterial(&stubMaterialRepo{}, &stubGroupProvider{}),
	)
	r := authedRequest(http.MethodGet, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Code != appMaterial.ErrCodeInsufficientPerms {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %s", resp.Code)
	}
}

// ── List handler tests ────────────────────────────────────────────────────────

func TestListMaterials_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/materials", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.List).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMaterials_MissingGroupId_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups//materials", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListMaterials_InvalidPage_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups/g1/materials?page=0", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error: %v", err)
	}
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestListMaterials_InvalidLimit_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups/g1/materials?limit=101", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListMaterials_ValidRequest_Returns200WithPagination(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups/g1/materials", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listMaterialsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Materials == nil {
		t.Error("expected materials array, got nil")
	}
	if resp.Pagination.ItemsPerPage != appMaterial.DefaultLimit {
		t.Errorf("expected itemsPerPage=%d, got %d", appMaterial.DefaultLimit, resp.Pagination.ItemsPerPage)
	}
}

func TestListMaterials_InvalidPinnedParam_Returns400(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodGet, "/groups/g1/materials?pinned=notabool", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Publish handler tests ─────────────────────────────────────────────────────

func TestPublishMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials/m1/publish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Publish).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPublishMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/publish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPublishMaterial_Success_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/publish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Status != "PUBLISHED" {
		t.Errorf("expected status PUBLISHED, got %q", resp.Status)
	}
}

// ── Unpublish handler tests ───────────────────────────────────────────────────

func TestUnpublishMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials/m1/unpublish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Unpublish).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUnpublishMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/unpublish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUnpublishMaterial_Success_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", false, nil,
		time.Now(), time.Now(), &now,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/unpublish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Status != "DRAFT" {
		t.Errorf("expected status DRAFT, got %q", resp.Status)
	}
}

// ── Pin handler tests ─────────────────────────────────────────────────────────

func TestPinMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials/m1/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Pin).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPinMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Pin)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPinMaterial_CannotPinDraft_Returns400(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Pin)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if resp.Code != domainMaterial.ErrCodeCannotPinDraft {
		t.Errorf("expected %s, got %s", domainMaterial.ErrCodeCannotPinDraft, resp.Code)
	}
}

func TestPinMaterial_Success_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", false, nil,
		time.Now(), time.Now(), &now,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Pin)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if !resp.Pinned {
		t.Error("expected pinned=true")
	}
}

// ── Unpin handler tests ───────────────────────────────────────────────────────

func TestUnpinMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials/m1/unpin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Unpin).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUnpinMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/unpin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpin)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUnpinMaterial_Success_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", true, &now,
		time.Now(), time.Now(), &now,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/unpin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpin)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp materialResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Pinned {
		t.Error("expected pinned=false")
	}
}

// ── Delete handler tests ──────────────────────────────────────────────────────

func TestDeleteMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := stubHandler()
	r := httptest.NewRequest(http.MethodDelete, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Delete).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteMaterial_NotFound_Returns404(t *testing.T) {
	h := stubHandler()
	r := authedRequest(http.MethodDelete, "/groups/g1/materials/missing", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteMaterial_Success_Returns204(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodDelete, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body for 204, got: %s", w.Body.String())
	}
}

func TestDeleteMaterial_Forbidden_Returns403WithAuthorId(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("other-author"),
		"Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := handlerWithRepo(&stubMaterialRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return mat, nil
		},
	})
	r := authedRequest(http.MethodDelete, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Error    string `json:"error"`
		AuthorID string `json:"authorId"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error response: %v (body: %s)", err, w.Body.String())
	}
	if resp.Error != appMaterial.ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR, got %s", resp.Error)
	}
	if resp.AuthorID != "other-author" {
		t.Errorf("expected authorId 'other-author', got %q", resp.AuthorID)
	}
}
