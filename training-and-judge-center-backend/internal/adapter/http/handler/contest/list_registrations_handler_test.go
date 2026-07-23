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

func newHandlerWithListRegistrations(uc *appcontest.ListContestRegistrationsUseCase) *Handler {
	return &Handler{listContestRegistrations: uc}
}

func defaultListRegistrationsUC() *appcontest.ListContestRegistrationsUseCase {
	return appcontest.NewListContestRegistrationsUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: true},
		&mockNicknameProvider{},
	)
}

func TestListRegistrations_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListRegistrations(defaultListRegistrationsUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/registrations", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.ListRegistrations).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListRegistrations_HappyPath_Returns200(t *testing.T) {
	uc := appcontest.NewListContestRegistrationsUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{
			listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
				return []*domainContest.ContestRegistration{
					domainContest.RestoreContestRegistration("r1", "c1", "u1", time.Now()),
				}, 1, nil
			},
		},
		&mockMemberProvider{isMember: true},
		&mockNicknameProvider{},
	)
	h := newHandlerWithListRegistrations(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/registrations", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListRegistrations)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Registrations []struct {
			Nickname     string `json:"nickname"`
			RegisteredAt string `json:"registeredAt"`
		} `json:"registrations"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Pagination.Total)
	}
	if len(resp.Registrations) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Registrations))
	}
	if resp.Registrations[0].Nickname == "" {
		t.Error("expected non-empty nickname")
	}
	if _, err := time.Parse(time.RFC3339, resp.Registrations[0].RegisteredAt); err != nil {
		t.Errorf("registeredAt is not RFC3339: %q", resp.Registrations[0].RegisteredAt)
	}
}

func TestListRegistrations_NonMember_Returns403(t *testing.T) {
	uc := appcontest.NewListContestRegistrationsUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: false},
		&mockNicknameProvider{},
	)
	h := newHandlerWithListRegistrations(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/registrations", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListRegistrations)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRegistrations_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewListContestRegistrationsUseCase(
		&mockRepoReturning{contest: nil},
		&mockRegistrationRepository{},
		&mockMemberProvider{isMember: true},
		&mockNicknameProvider{},
	)
	h := newHandlerWithListRegistrations(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/missing/registrations", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListRegistrations)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
