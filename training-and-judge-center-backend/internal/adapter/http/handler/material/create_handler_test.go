package material

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithCreate(uc *appMaterial.CreateMaterialUseCase) *Handler {
	return &Handler{createMaterial: uc}
}

func defaultCreateUC() *appMaterial.CreateMaterialUseCase {
	return appMaterial.NewCreateMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubGroupMemberProvider{}, &stubAuthorProvider{},
	)
}

func TestCreateMaterial_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/materials", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Create).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateMaterial_EmptyGroupId_Returns400(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())
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
	h := newHandlerWithCreate(defaultCreateUC())
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
	h := newHandlerWithCreate(defaultCreateUC())
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
	h := newHandlerWithCreate(appMaterial.NewCreateMaterialUseCase(
		&stubMaterialRepo{}, &stubGroupProvider{}, &stubNotLeadMemberProvider{}, &stubAuthorProvider{},
	))
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
	if resp.Code != appMaterial.ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %s", resp.Code)
	}
}
