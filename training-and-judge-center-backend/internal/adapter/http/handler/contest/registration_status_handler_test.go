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
)

func newHandlerWithStatus(uc *appcontest.GetRegistrationStatusUseCase) *Handler {
	return &Handler{getRegistrationStatus: uc}
}

func defaultStatusUC() *appcontest.GetRegistrationStatusUseCase {
	return appcontest.NewGetRegistrationStatusUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
	)
}

func TestGetRegistrationStatus_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithStatus(defaultStatusUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/register/status", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetRegistrationStatus).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetRegistrationStatus_NotRegistered_Returns200(t *testing.T) {
	h := newHandlerWithStatus(defaultStatusUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/register/status", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetRegistrationStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Registered   bool    `json:"registered"`
		RegisteredAt *string `json:"registeredAt"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Registered {
		t.Error("expected registered=false")
	}
	if resp.RegisteredAt != nil {
		t.Error("expected no registeredAt when not registered")
	}
}

func TestGetRegistrationStatus_Registered_Returns200WithTimestamp(t *testing.T) {
	uc := appcontest.NewGetRegistrationStatusUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{
			findByContestAndUser: func(_ context.Context, _, _ string) (*domainContest.ContestRegistration, error) {
				return domainContest.RestoreContestRegistration("r1", "c1", "u1", time.Now()), nil
			},
		},
	)
	h := newHandlerWithStatus(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/register/status", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetRegistrationStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Registered   bool    `json:"registered"`
		RegisteredAt *string `json:"registeredAt"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if !resp.Registered {
		t.Error("expected registered=true")
	}
	if resp.RegisteredAt == nil {
		t.Fatal("expected non-nil registeredAt")
	}
	if _, err := time.Parse(time.RFC3339, *resp.RegisteredAt); err != nil {
		t.Errorf("registeredAt is not RFC3339: %q", *resp.RegisteredAt)
	}
}

func TestGetRegistrationStatus_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewGetRegistrationStatusUseCase(
		&mockRepoReturning{contest: nil},
		&mockRegistrationRepository{},
	)
	h := newHandlerWithStatus(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/missing/register/status", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetRegistrationStatus)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
