package material

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newHandlerWithDelete(uc *appMaterial.DeleteMaterialUseCase) *Handler {
	return &Handler{deleteMaterial: uc}
}

func defaultDeleteUC() *appMaterial.DeleteMaterialUseCase {
	return appMaterial.NewDeleteMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{},
	)
}

func TestDeleteMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithDelete(defaultDeleteUC())
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
	h := newHandlerWithDelete(defaultDeleteUC())
	r := authedRequest(http.MethodDelete, "/groups/g1/materials/missing", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteMaterial_ValidRequest_Returns204(t *testing.T) {
	now := time.Now()
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "DRAFT", false, nil,
		now, now, nil,
	)
	h := newHandlerWithDelete(appMaterial.NewDeleteMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{},
	))

	r := authedRequest(http.MethodDelete, "/groups/g1/materials/m1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "m1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
