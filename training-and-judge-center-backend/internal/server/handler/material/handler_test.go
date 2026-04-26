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

type stubGroupMemberProvider struct{}

func (s *stubGroupMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func stubHandler() *Handler {
	return NewHandler(
		appMaterial.NewCreateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}),
		appMaterial.NewUpdateMaterial(&stubMaterialRepo{}, &stubGroupProvider{}),
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

func TestCreateMaterial_MissingGroupId_Returns400(t *testing.T) {
	h := stubHandler()
	body, _ := json.Marshal(map[string]string{"title": "Hello"})
	r := authedRequest(http.MethodPost, "/groups//materials", body)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	json.NewDecoder(w.Body).Decode(&resp)
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
	json.NewDecoder(w.Body).Decode(&resp)
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

func TestUpdateMaterial_ValidRequest_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Old Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	repo := &stubMaterialRepo{}
	h := NewHandler(
		appMaterial.NewCreateMaterial(repo, &stubGroupProvider{}, &stubGroupMemberProvider{}),
		appMaterial.NewUpdateMaterial(&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		}, &stubGroupProvider{}),
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
	json.NewDecoder(w.Body).Decode(&resp)
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
	json.NewDecoder(w.Body).Decode(&resp)
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
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}
