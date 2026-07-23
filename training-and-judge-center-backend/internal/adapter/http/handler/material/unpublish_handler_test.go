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

func newHandlerWithUnpublish(uc *appMaterial.UnpublishMaterialUseCase) *Handler {
	return &Handler{unpublishMaterial: uc}
}

func defaultUnpublishUC() *appMaterial.UnpublishMaterialUseCase {
	return appMaterial.NewUnpublishMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{},
	)
}

func TestUnpublishMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUnpublish(defaultUnpublishUC())
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
	h := newHandlerWithUnpublish(defaultUnpublishUC())
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/unpublish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUnpublishMaterial_ValidRequest_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", false, nil,
		now, now, &now,
	)
	h := newHandlerWithUnpublish(appMaterial.NewUnpublishMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubAuthorProvider{},
	))

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
