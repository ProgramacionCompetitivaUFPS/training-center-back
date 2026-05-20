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
		&mockGroupProvider{},
		&mockMemberProvider{isLead: false, isMember: true},
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
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "not found")
	}
	return s.contest, nil
}
func (s *mockRepoReturning) Delete(_ context.Context, _ string) error { return nil }
func (s *mockRepoReturning) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
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

func TestRegister_HappyPath_Returns201(t *testing.T) {
	h := newHandlerWithRegister(defaultRegisterUC())
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp registerResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.RegisteredAt == "" {
		t.Error("expected non-empty registeredAt")
	}
}

func TestRegister_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: nil},
		&mockRegistrationRepository{},
		&mockGroupProvider{},
		&mockMemberProvider{isMember: true},
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
		&mockGroupProvider{},
		&mockMemberProvider{isLead: false, isMember: false},
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

func TestRegister_AlreadyRegistered_Returns409(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{
			existsByContestAndUser: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		},
		&mockGroupProvider{},
		&mockMemberProvider{isMember: true},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_RegistrationClosed_Returns409(t *testing.T) {
	uc := appcontest.NewRegisterToContestUseCase(
		&mockRepoReturning{contest: finishedContest()},
		&mockRegistrationRepository{},
		&mockGroupProvider{},
		&mockMemberProvider{isMember: true},
	)
	h := newHandlerWithRegister(uc)
	r := authedRequest(http.MethodPost, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Register)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
