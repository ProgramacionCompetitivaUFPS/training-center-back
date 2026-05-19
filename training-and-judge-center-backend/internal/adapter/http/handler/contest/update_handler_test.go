package contest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithUpdate(uc *appcontest.UpdateContestUseCase) *Handler {
	return &Handler{updateContest: uc}
}

// stubUpdateRepo returns a valid contest from FindByID; all writes are no-ops.
type stubUpdateRepo struct{}

func (s *stubUpdateRepo) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *stubUpdateRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *stubUpdateRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Old Name"),
		nil,
		time.Now().Add(24*time.Hour),
		time.Now().Add(29*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		[]domainContest.ContestProblem{},
		time.Now().Add(-time.Hour),
		nil,
	), nil
}
func (s *stubUpdateRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubUpdateRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

func defaultUpdateUC() *appcontest.UpdateContestUseCase {
	return appcontest.NewUpdateContestUseCase(
		&stubUpdateRepo{},
		&stubGroupProvider{},
		&stubMemberProvider{isLead: true},
		&stubProblemProvider{},
		&stubOwnerProvider{},
		&stubTransactionManager{},
	)
}

func TestUpdateContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := httptest.NewRequest(http.MethodPut, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Update).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateContest_MissingPathParams_Returns400(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := authedRequest(http.MethodPut, "/groups//contests/", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateContest_InvalidBody_Returns400(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := authedRequest(http.MethodPut, "/groups/g1/contests/c1", []byte("not-json"))
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

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

func TestUpdateContest_ValidRequest_Returns200(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())

	newName := "Updated Name"
	body, _ := json.Marshal(map[string]interface{}{"name": newName})
	r := authedRequest(http.MethodPut, "/groups/g1/contests/c1", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp contestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Name != newName {
		t.Errorf("expected name %q, got %q", newName, resp.Name)
	}
}

func TestUpdateContest_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewUpdateContestUseCase(
		&stubContestRepo{}, // FindByID returns 404 by default
		&stubGroupProvider{},
		&stubMemberProvider{isLead: true},
		&stubProblemProvider{},
		&stubOwnerProvider{},
		&stubTransactionManager{},
	)
	h := newHandlerWithUpdate(uc)

	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	r := authedRequest(http.MethodPut, "/groups/g1/contests/missing", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
