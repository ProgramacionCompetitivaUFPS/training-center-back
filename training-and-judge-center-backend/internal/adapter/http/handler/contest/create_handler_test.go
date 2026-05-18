package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithCreate(uc *appcontest.CreateContestUseCase) *Handler {
	return &Handler{createContest: uc}
}

func defaultCreateUC() *appcontest.CreateContestUseCase {
	return appcontest.NewCreateContestUseCase(
		&stubContestRepo{},
		&stubGroupProvider{},
		&stubMemberProvider{isLead: true},
		&stubProblemProvider{},
		&stubOwnerProvider{},
	)
}

func TestCreateContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/contests", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Create).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateContest_MissingGroupId_Returns400(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())
	r := authedRequest(http.MethodPost, "/groups//contests", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateContest_InvalidBody_Returns400(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())
	r := authedRequest(http.MethodPost, "/groups/g1/contests", []byte("not-json"))
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error: %v", err)
	}
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestCreateContest_ValidRequest_Returns201(t *testing.T) {
	h := newHandlerWithCreate(defaultCreateUC())

	body, _ := json.Marshal(map[string]interface{}{
		"name":      "My Contest",
		"startTime": time.Now().Add(24 * time.Hour),
		"endTime":   time.Now().Add(29 * time.Hour),
	})
	r := authedRequest(http.MethodPost, "/groups/g1/contests", body)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp contestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Name != "My Contest" {
		t.Errorf("expected name 'My Contest', got %q", resp.Name)
	}
}

func TestCreateContest_ForbiddenIfNotLead_Returns403(t *testing.T) {
	uc := appcontest.NewCreateContestUseCase(
		&stubContestRepo{},
		&stubGroupProvider{},
		&stubMemberProvider{isLead: false},
		&stubProblemProvider{},
		&stubOwnerProvider{},
	)
	h := newHandlerWithCreate(uc)

	body, _ := json.Marshal(map[string]interface{}{
		"name":      "My Contest",
		"startTime": time.Now().Add(24 * time.Hour),
		"endTime":   time.Now().Add(29 * time.Hour),
	})
	r := authedRequest(http.MethodPost, "/groups/g1/contests", body)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
