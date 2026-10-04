package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestListProblems_SearchLength(t *testing.T) {
	tests := []struct {
		name       string
		search     string
		wantStatus int
	}{
		{"200 characters", strings.Repeat("a", 200), http.StatusOK},
		{"201 characters", strings.Repeat("a", 201), http.StatusBadRequest},
		{"200 multibyte characters", strings.Repeat("ñ", 200), http.StatusOK},
		{"201 multibyte characters", strings.Repeat("ñ", 201), http.StatusBadRequest},
		{"wildcards and quotes", `100%_\'"`, http.StatusOK},
	}

	h := newHandlerWithListProblems(&mockProblemRepo{}, &mockUserProviderH{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := authedRequest(http.MethodGet, "/problems?search="+url.QueryEscape(tt.search), nil)
			w := httptest.NewRecorder()

			wrapAuth(http.HandlerFunc(h.ListProblems)).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus != http.StatusBadRequest {
				return
			}
			var resp apperror.AppError
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("could not decode response: %v", err)
			}
			if resp.Code != apperror.ErrCodeValidationError || len(resp.Details) != 1 || resp.Details[0].Field != "search" {
				t.Errorf("expected VALIDATION_ERROR on field search, got %+v", resp)
			}
		})
	}
}
