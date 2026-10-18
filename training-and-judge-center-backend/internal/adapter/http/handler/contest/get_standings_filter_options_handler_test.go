package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
)

func TestGetStandingsFilterOptions_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetStandings(defaultGetStandingsUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/standings/filters", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetStandingsFilterOptions).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetStandingsFilterOptions_HappyPath_ReturnsDistinctValues(t *testing.T) {
	cached := &appcontest.CachedStandings{
		Profiles: map[string]*appcontest.ParticipantProfile{
			"u1": {ID: "u1", Country: "Colombia", City: "Bogotá", Institution: "UFPS"},
			"u2": {ID: "u2", Country: "Argentina", City: "Bogotá", Institution: ""},
		},
		LastUpdated: time.Now(),
	}
	uc := appcontest.NewGetStandingsUseCase(
		&mockStandingsContestRepo{contest: activeContestForStandings()},
		&mockRegistrationRepository{},
		&mockSubmissionProvider{},
		&mockTeamParticipantProvider{},
		&mockParticipantProfileProvider{},
		&mockTeamDisplayProvider{},
		&mockProblemProvider{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockStandingsCache{data: cached},
		30*time.Second,
	)
	h := newHandlerWithGetStandings(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings/filters", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandingsFilterOptions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsFilterOptionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Countries) != 2 || resp.Countries[0] != "Argentina" || resp.Countries[1] != "Colombia" {
		t.Errorf("countries = %v, want sorted [Argentina Colombia]", resp.Countries)
	}
	if len(resp.Cities) != 1 || resp.Cities[0] != "Bogotá" {
		t.Errorf("cities = %v, want deduped [Bogotá]", resp.Cities)
	}
	if len(resp.Institutions) != 1 || resp.Institutions[0] != "UFPS" {
		t.Errorf("institutions = %v, want [UFPS]", resp.Institutions)
	}
}
