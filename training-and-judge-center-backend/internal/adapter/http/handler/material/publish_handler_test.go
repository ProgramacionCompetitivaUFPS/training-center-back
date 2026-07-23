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

func newHandlerWithPublish(uc *appMaterial.PublishMaterialUseCase) *Handler {
	return &Handler{publishMaterial: uc}
}

func defaultPublishUC() *appMaterial.PublishMaterialUseCase {
	return appMaterial.NewPublishMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{},
	)
}

func TestPublishMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithPublish(defaultPublishUC())
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
	h := newHandlerWithPublish(defaultPublishUC())
	r := authedRequest(http.MethodPost, "/groups/g1/materials/missing/publish", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("materialId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPublishMaterial_ValidRequest_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := newHandlerWithPublish(appMaterial.NewPublishMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubAuthorProvider{},
	))

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
