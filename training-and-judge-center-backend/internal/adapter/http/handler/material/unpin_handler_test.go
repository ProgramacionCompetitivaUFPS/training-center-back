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

func newHandlerWithUnpin(uc *appMaterial.UnpinMaterialUseCase) *Handler {
	return &Handler{unpinMaterial: uc}
}

func defaultUnpinUC() *appMaterial.UnpinMaterialUseCase {
	return appMaterial.NewUnpinMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	)
}

func TestUnpinMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUnpin(defaultUnpinUC())
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
	h := newHandlerWithUnpin(defaultUnpinUC())
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/unpin", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpin)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUnpinMaterial_ValidRequest_Returns200(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "PUBLISHED", true, nil,
		now, now, &now,
	)
	h := newHandlerWithUnpin(appMaterial.NewUnpinMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	))

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
