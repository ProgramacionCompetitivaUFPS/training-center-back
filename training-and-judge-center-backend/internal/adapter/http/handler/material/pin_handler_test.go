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
)

func newHandlerWithPin(uc *appMaterial.PinMaterialUseCase) *Handler {
	return &Handler{pinMaterial: uc}
}

func defaultPinUC() *appMaterial.PinMaterialUseCase {
	return appMaterial.NewPinMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	)
}

func TestPinMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithPin(defaultPinUC())
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
	h := newHandlerWithPin(defaultPinUC())
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Pin)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPinMaterial_ValidRequest_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", false, nil,
		now, now, &now,
	)
	h := newHandlerWithPin(appMaterial.NewPinMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	))

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

func TestPinMaterial_NonLeadNonAuthor_Returns403(t *testing.T) {
	now := time.Now()
	// Author is "other", but mock user is "u1" — so u1 is neither lead nor author
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("other"),
		"Title", "", nil, "PUBLISHED", false, nil,
		now, now, &now,
	)
	h := newHandlerWithPin(appMaterial.NewPinMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubNotLeadMemberProvider{}, &stubAuthorProvider{},
	))

	r := authedRequest(http.MethodPost, "/groups/g1/materials/m1/pin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Pin)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
