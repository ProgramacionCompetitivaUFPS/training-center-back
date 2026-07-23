package material

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithGet(uc *appMaterial.GetMaterialUseCase) *Handler {
	return &Handler{getMaterial: uc}
}

func defaultGetUC() *appMaterial.GetMaterialUseCase {
	return appMaterial.NewGetMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	)
}

func TestGetMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGet(defaultGetUC())
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
	h := newHandlerWithGet(defaultGetUC())
	r := authedRequest(http.MethodGet, "/groups//materials/", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

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

func TestGetMaterial_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithGet(defaultGetUC())
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
		"My Material", "", nil, "PUBLISHED", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := newHandlerWithGet(appMaterial.NewGetMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupVisibilityProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	))

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
	if resp.Title != "My Material" {
		t.Errorf("expected title 'My Material', got %q", resp.Title)
	}
}

func TestGetMaterial_NonMember_Returns403(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Secret", "", nil, "PUBLISHED", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := newHandlerWithGet(appMaterial.NewGetMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubNotVisibleGroupVisibilityProvider{}, &stubNonMemberProvider{}, &stubAuthorProvider{},
	))

	r := authedRequest(http.MethodGet, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
