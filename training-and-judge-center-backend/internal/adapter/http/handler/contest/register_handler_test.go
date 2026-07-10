package contest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithRegister(uc *appcontest.RegisterToContestUseCase) *Handler {
	return &Handler{registerToContest: uc}
}

func defaultRegisterUC() *appcontest.RegisterToContestUseCase {
	repo := &mockRepoReturning{contest: scheduledContest()}
	return appcontest.NewRegisterToContestUseCase(
		repo,
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: true},
		&noopTeamSelectionChecker{},
	)
}

// mockRepoReturning returns a fixed contest from FindByID and acts as a no-op for other ops.
type mockRepoReturning struct {
	contest *domainContest.Contest
}

func (s *mockRepoReturning) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *mockRepoReturning) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *mockRepoReturning) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	if s.contest == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}
	return s.contest, nil
}
func (s *mockRepoReturning) Delete(_ context.Context, _ string) error { return nil }
func (s *mockRepoReturning) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}
func (s *mockRepoReturning) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

func TestRegister_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithRegister(defaultRegisterUC())
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Register).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRegister_HappyPath_Returns204(t *testing.T) {
	h := newHandlerWithRegister(defaultRegisterUC())
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestRegister_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: nil},
		&mockRegistrationRepository{},
		&mockMemberProvider{isMember: true},
		&noopTeamSelectionChecker{},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/missing/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_NonMember_Returns403(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: false},
		&noopTeamSelectionChecker{},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_AlreadyRegistered_Returns204(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{
			existsByContestAndUserFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		},
		&mockMemberProvider{isMember: true},
		&noopTeamSelectionChecker{},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 (idempotent), got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_ContestAlreadyStarted_Returns400(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: finishedContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isMember: true},
		&noopTeamSelectionChecker{},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var errResp struct {
		Code string `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("could not decode error body: %v", err)
	}
	if errResp.Code != domainContest.ErrCodeContestAlreadyStarted {
		t.Errorf("expected error code %q, got %q", domainContest.ErrCodeContestAlreadyStarted, errResp.Code)
	}
}
