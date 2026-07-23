package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdateJoinRequest_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/groups/g1/requests/r1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateJoinRequest_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests/r1", `{bad json}`)
	r.Method = "PATCH"
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateJoinRequest_InvalidStatusReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests/r1", `{"status":"PENDING"}`)
	r.Method = "PATCH"
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}
