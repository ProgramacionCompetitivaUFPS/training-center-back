package group

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
	"encoding/json"
)

func TestListGroups_NonIntegerPageReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestListGroups_NonIntegerLimitReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?limit=xyz"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

// â”€â”€ buildPagination tests â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBuildPagination_FirstPageOfMany_HasNextNoPrev(t *testing.T) {
	p := buildPagination(30, 1, 3, 10)
	if !p.HasNextPage {
		t.Error("expected HasNextPage=true on page 1 of 3")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false on page 1")
	}
}

func TestBuildPagination_LastPage_HasPrevNoNext(t *testing.T) {
	p := buildPagination(30, 3, 3, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false on last page")
	}
	if !p.HasPrevPage {
		t.Error("expected HasPrevPage=true on page 3 of 3")
	}
}

func TestBuildPagination_MiddlePage_HasBoth(t *testing.T) {
	p := buildPagination(30, 2, 3, 10)
	if !p.HasNextPage {
		t.Error("expected HasNextPage=true on middle page")
	}
	if !p.HasPrevPage {
		t.Error("expected HasPrevPage=true on middle page")
	}
}

func TestBuildPagination_SinglePage_HasNeither(t *testing.T) {
	p := buildPagination(5, 1, 1, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false on single page")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false on single page")
	}
}

func TestBuildPagination_ZeroResults_HasNeither(t *testing.T) {
	p := buildPagination(0, 1, 0, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false with 0 results")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false with 0 results")
	}
}
