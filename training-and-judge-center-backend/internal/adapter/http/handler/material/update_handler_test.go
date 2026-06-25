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

func newHandlerWithUpdate(uc *appMaterial.UpdateMaterialUseCase) *Handler {
	return &Handler{updateMaterial: uc}
}

func defaultUpdateUC() *appMaterial.UpdateMaterialUseCase {
	return appMaterial.NewUpdateMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubAuthorProvider{},
	)
}

func TestUpdateMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
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
	h := newHandlerWithUpdate(defaultUpdateUC())
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
	h := newHandlerWithUpdate(defaultUpdateUC())
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

func TestUpdateMaterial_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
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

func TestUpdateMaterial_ValidRequest_Returns200(t *testing.T) {
	mat := domainMaterial.RestoreMaterial(
		"m1", "g1", shared.RestoreUserID("u1"),
		"Old Title", "", nil, "DRAFT", false, nil,
		time.Now(), time.Now(), nil,
	)
	h := newHandlerWithUpdate(appMaterial.NewUpdateMaterialUseCase(
		&stubMaterialRepo{
			findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
				return mat, nil
			},
		},
		&stubGroupProvider{}, &stubAuthorProvider{},
	))

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
