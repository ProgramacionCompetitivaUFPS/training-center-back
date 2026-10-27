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
)

// ── handler-scoped mocks ──────────────────────────────────────────────────────

type mockStandingsContestRepo struct {
	contest *domainContest.Contest
}

func (m *mockStandingsContestRepo) Create(_ context.Context, _ *domainContest.Contest) error {
	return nil
}
func (m *mockStandingsContestRepo) Update(_ context.Context, _ *domainContest.Contest) error {
	return nil
}
func (m *mockStandingsContestRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return m.contest, nil
}
func (m *mockStandingsContestRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockStandingsContestRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}
func (m *mockStandingsContestRepo) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

type mockStandingsCache struct {
	data *appcontest.CachedStandings
}

func (m *mockStandingsCache) Get(_ context.Context, _ string) (*appcontest.CachedStandings, error) {
	return m.data, nil
}
func (m *mockStandingsCache) Set(_ context.Context, _ string, _ *appcontest.CachedStandings) error {
	return nil
}
func (m *mockStandingsCache) AcquireRefreshLock(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (m *mockStandingsCache) ReleaseRefreshLock(_ context.Context, _ string) error { return nil }
func (m *mockStandingsCache) Invalidate(_ context.Context, _ string) error         { return nil }

type mockSubmissionProvider struct{}

func (m *mockSubmissionProvider) ListByContest(_ context.Context, _ string) ([]appcontest.ContestSubmissionData, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newHandlerWithGetStandings(uc *appcontest.GetStandingsUseCase) *Handler {
	return &Handler{getStandings: uc}
}

func activeContestForStandings() *domainContest.Contest {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(2*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	)
}

func defaultGetStandingsUC() *appcontest.GetStandingsUseCase {
	accepted := time.Now().Add(-60 * time.Minute)
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", Problems: map[string]domainContest.ProblemAttempt{
				"A": {AcceptedAt: &accepted},
			}},
		},
		LastUpdated: time.Now(),
	}
	return appcontest.NewGetStandingsUseCase(
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
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetStandings_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetStandings(defaultGetStandingsUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/standings", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetStandings).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetStandings_HappyPath_Returns200(t *testing.T) {
	h := newHandlerWithGetStandings(defaultGetStandingsUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Standings) != 1 {
		t.Errorf("standings=%d, want 1", len(resp.Standings))
	}
	if resp.Standings[0].ProblemsSolved != 1 {
		t.Errorf("problemsSolved=%d, want 1", resp.Standings[0].ProblemsSolved)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total=%d, want 1", resp.Pagination.Total)
	}
	if resp.Contest.Status != "ACTIVE" {
		t.Errorf("contestStatus=%q, want ACTIVE", resp.Contest.Status)
	}
}

func TestGetStandings_Pagination_Returns200(t *testing.T) {
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u2", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u3", Problems: map[string]domainContest.ProblemAttempt{}},
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

	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings?page=2&limit=1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("total=%d, want 3", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 3 {
		t.Errorf("totalPages=%d, want 3", resp.Pagination.TotalPages)
	}
	if len(resp.Standings) != 1 {
		t.Errorf("page standings=%d, want 1", len(resp.Standings))
	}
}

func TestGetStandings_CountryQueryParamFiltersEntries(t *testing.T) {
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u2", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{}},
		},
		Profiles: map[string]*appcontest.ParticipantProfile{
			"u1": {ID: "u1", Country: "colombia"},
			"u2": {ID: "u2", Country: "mexico"},
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

	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings?country=colombia", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Standings) != 1 || resp.Standings[0].Participant.ID != "u1" {
		t.Fatalf("expected only u1 to survive ?country=colombia, got %+v", resp.Standings)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total=%d, want 1", resp.Pagination.Total)
	}
}

// TestGetStandings_FullContractShape_Returns200 pins the real API contract
// (COM-09): contest summary, ordered problem headers, and per-entry
// participant display (individual profile fields, team name + members),
// with problems reshaped from the internal map into the ordered-by-position
// array the frontend expects, including a NOT_ATTEMPTED problem no one touched.
func TestGetStandings_FullContractShape_Returns200(t *testing.T) {
	contest := domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(2*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, true, // showTeamMembers=true
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("MIXED"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{
			domainContest.RestoreContestProblem("cp1", "prob-A", 1),
			domainContest.RestoreContestProblem("cp2", "prob-B", 2),
		},
		time.Now(),
		nil,
	)

	accepted := time.Now().Add(-90 * time.Minute) // 30 minutes after contest start
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{
				ContestantID: "u1", ParticipantType: "INDIVIDUAL",
				Problems: map[string]domainContest.ProblemAttempt{
					"prob-A": {AcceptedAt: &accepted},
				},
			},
			{
				ContestantID: "team-1", ParticipantType: "TEAM",
				Problems: map[string]domainContest.ProblemAttempt{
					"prob-B": {WrongAttemptTimes: []time.Time{time.Now().Add(-70 * time.Minute)}},
				},
			},
		},
		TeamMembers: map[string][]string{"team-1": {"m1", "m2"}},
		TeamNames:   map[string]string{"team-1": "Code Ninjas"},
		Profiles: map[string]*appcontest.ParticipantProfile{
			"u1": {ID: "u1", Nickname: "luisadmin", Name: "Luis Admin", Country: "Colombia", City: "Bucaramanga", Institution: "UFPS"},
			"m1": {ID: "m1", Nickname: "carloscp"},
			"m2": {ID: "m2", Nickname: "anagarcia"},
		},
		LastUpdated: time.Now(),
	}

	uc := appcontest.NewGetStandingsUseCase(
		&mockStandingsContestRepo{contest: contest},
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

	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Contest.ID != "c1" || resp.Contest.Status != "ACTIVE" {
		t.Errorf("contest=%+v, want id=c1 status=ACTIVE", resp.Contest)
	}
	if len(resp.Problems) != 2 || resp.Problems[0].Position != 1 || resp.Problems[1].Position != 2 {
		t.Fatalf("problems=%+v, want two ordered headers", resp.Problems)
	}
	if resp.Problems[0].Slug == "" || resp.Problems[0].Title == "" {
		t.Errorf("problem header missing slug/title: %+v", resp.Problems[0])
	}

	if len(resp.Standings) != 2 {
		t.Fatalf("standings=%d, want 2", len(resp.Standings))
	}

	var individual, team *standingEntry
	for i := range resp.Standings {
		switch resp.Standings[i].Participant.ID {
		case "u1":
			individual = &resp.Standings[i]
		case "team-1":
			team = &resp.Standings[i]
		}
	}
	if individual == nil || team == nil {
		t.Fatalf("expected both u1 and team-1 in standings, got %+v", resp.Standings)
	}

	// individual: full profile enrichment, and per-problem array in position order
	p := individual.Participant
	if p.Type != "INDIVIDUAL" || p.DisplayName != "luisadmin" || p.Nickname != "luisadmin" || p.Name != "Luis Admin" {
		t.Errorf("individual participant=%+v, want luisadmin display fields", p)
	}
	if p.Country == nil || *p.Country != "Colombia" {
		t.Errorf("individual country=%v, want Colombia", p.Country)
	}
	if len(individual.Problems) != 2 {
		t.Fatalf("individual problems=%d, want 2", len(individual.Problems))
	}
	if individual.Problems[0].Status != "ACCEPTED" || individual.Problems[0].Time == nil || *individual.Problems[0].Time != 30 {
		t.Errorf("problem A=%+v, want ACCEPTED at minute 30", individual.Problems[0])
	}
	if individual.Problems[1].Status != "NOT_ATTEMPTED" {
		t.Errorf("problem B for u1=%+v, want NOT_ATTEMPTED", individual.Problems[1])
	}

	// team: name resolved, members exposed (showTeamMembers=true), no location fields
	tp := team.Participant
	if tp.Type != "TEAM" || tp.DisplayName != "Code Ninjas" {
		t.Errorf("team participant=%+v, want displayName=Code Ninjas", tp)
	}
	if len(tp.Members) != 2 || tp.Members[0] != "anagarcia" || tp.Members[1] != "carloscp" {
		t.Errorf("team members=%v, want sorted [anagarcia carloscp]", tp.Members)
	}
	if tp.Country != nil {
		t.Errorf("team country=%v, want nil (teams have no location)", tp.Country)
	}
	if team.Problems[1].Status != "WRONG_ANSWER" || team.Problems[1].Attempts != 1 {
		t.Errorf("problem B for team=%+v, want WRONG_ANSWER with 1 attempt", team.Problems[1])
	}

	if resp.Filters.Country != nil {
		t.Errorf("filters.country=%v, want nil (no filter applied)", resp.Filters.Country)
	}
	if resp.Filters.FilteredTotal != resp.Pagination.Total {
		t.Errorf("filteredTotal=%d, want %d (mirrors pagination.total)", resp.Filters.FilteredTotal, resp.Pagination.Total)
	}
}
