package material

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithList(uc *appMaterial.ListMaterialsUseCase) *Handler {
	return &Handler{listMaterials: uc}
}

func defaultListUC() *appMaterial.ListMaterialsUseCase {
	return appMaterial.NewListMaterialsUseCase(
		&stubMaterialRepo{}, &stubGroupVisibilityProvider{}, &stubGroupMemberProvider{},
		&stubAuthorProvider{}, &stubAuthorIDProvider{},
	)
}

func TestListMaterials_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/materials", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.List).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMaterials_EmptyGroupId_Returns400(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups//materials", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

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

func TestListMaterials_InvalidPage_Returns400(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups/g1/materials?page=abc", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListMaterials_InvalidPublishedFrom_Returns400(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups/g1/materials?publishedFrom=not-a-date", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListMaterials_ValidRequest_Returns200(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
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
		t.Error("expected non-nil materials slice")
	}
}

func TestListMaterials_WithSearchQuery_Returns200(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups/g1/materials?q=hello&sort=relevance", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
