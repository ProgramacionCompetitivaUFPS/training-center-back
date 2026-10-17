package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTags_ReturnsSortedAllowedTags(t *testing.T) {
	h := &Handler{settings: newTestSettings()} // allowedTags = {"dp"}

	r := authedRequest(http.MethodGet, "/problems/tags", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListTags)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listProblemTagsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Tags) != 1 || resp.Tags[0] != "dp" {
		t.Errorf("expected [\"dp\"], got %v", resp.Tags)
	}
}

func TestListTags_RequiresAuth(t *testing.T) {
	h := &Handler{settings: newTestSettings()}

	r := httptest.NewRequest(http.MethodGet, "/problems/tags", nil)
	w := httptest.NewRecorder()

	h.ListTags(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
