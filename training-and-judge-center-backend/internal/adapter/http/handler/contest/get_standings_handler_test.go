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
			domainContest.RestoreContestProblem("cp1", "prob-A", 1, nil),
			domainContest.RestoreContestProblem("cp2", "prob-B", 2, nil),
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

func TestGetStandings_BoardFields_ColorCountsAndFirstSolve(t *testing.T) {
	color := "#aabbcc"
	contest := domainContest.RestoreContest(
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
		[]domainContest.ContestProblem{
			domainContest.RestoreContestProblem("cp1", "prob-A", 1, &color),
			domainContest.RestoreContestProblem("cp2", "prob-B", 2, nil),
		},
		time.Now(),
		nil,
	)
	early, late := time.Now().Add(-100*time.Minute), time.Now().Add(-60*time.Minute)
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
				"prob-A": {AcceptedAt: &late},
			}},
			{ContestantID: "u2", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
				"prob-A": {AcceptedAt: &early},
				"prob-B": {WrongAttemptTimes: []time.Time{late}},
			}},
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
	var raw struct {
		Problems  []map[string]any `json:"problems"`
		Standings []struct {
			Participant struct {
				ID string `json:"id"`
			} `json:"participant"`
			Problems []map[string]any `json:"problems"`
		} `json:"standings"`
	}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got := raw.Problems[0]["color"]; got != "#aabbcc" {
		t.Errorf("header A color = %v, want #aabbcc", got)
	}
	if got, present := raw.Problems[1]["color"]; !present || got != nil {
		t.Errorf("header B color = %v (present=%v), want an explicit null", got, present)
	}
	if raw.Problems[0]["accepted"] != float64(2) || raw.Problems[0]["tried"] != float64(2) {
		t.Errorf("header A accepted/tried = %v/%v, want 2/2", raw.Problems[0]["accepted"], raw.Problems[0]["tried"])
	}
	if raw.Problems[1]["accepted"] != float64(0) || raw.Problems[1]["tried"] != float64(1) {
		t.Errorf("header B accepted/tried = %v/%v, want 0/1", raw.Problems[1]["accepted"], raw.Problems[1]["tried"])
	}

	firstSolve := map[string]any{}
	for _, e := range raw.Standings {
		firstSolve[e.Participant.ID] = e.Problems[0]["isFirstSolve"]
		if _, present := e.Problems[1]["isFirstSolve"]; !present {
			t.Errorf("%s: isFirstSolve must be present on every cell, including unsolved ones", e.Participant.ID)
		}
	}
	if firstSolve["u2"] != true || firstSolve["u1"] != false {
		t.Errorf("isFirstSolve on A = %v, want u2 true and u1 false", firstSolve)
	}
}

func finishedPostStandingsContest() *domainContest.Contest {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		time.Now().Add(-4*time.Hour),
		time.Now().Add(-time.Hour),
		domainContest.RestorePenalty(20),
		0, true, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{
			domainContest.RestoreContestProblem("cp1", "prob-A", 1, nil),
		},
		time.Now(),
		nil,
	)
}

func postStandingsUC(official, post []domainContest.ParticipantStanding) *appcontest.GetStandingsUseCase {
	cached := &appcontest.CachedStandings{Participants: official, PostParticipants: post, LastUpdated: time.Now()}
	return appcontest.NewGetStandingsUseCase(
		&mockStandingsContestRepo{contest: finishedPostStandingsContest()},
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

func serveStandings(t *testing.T, uc *appcontest.GetStandingsUseCase, target string) *httptest.ResponseRecorder {
	t.Helper()
	h := newHandlerWithGetStandings(uc)
	r := authedRequest(http.MethodGet, target, nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)
	return w
}

func TestGetStandings_InvalidPhase_Returns400(t *testing.T) {
	w := serveStandings(t, postStandingsUC(nil, nil), "/groups/g1/contests/c1/standings?phase=bogus")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "BAD_REQUEST" {
		t.Errorf("error = %v, want BAD_REQUEST in key \"error\"", body["error"])
	}
}

func TestGetStandings_PhaseOmitted_ListsOfficialThenPostRows(t *testing.T) {
	early := time.Now().Add(-2 * time.Hour)
	late := time.Now().Add(-30 * time.Minute)
	w := serveStandings(t,
		postStandingsUC(
			[]domainContest.ParticipantStanding{{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &early}}}},
			[]domainContest.ParticipantStanding{{ContestantID: "p1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &late}}}},
		),
		"/groups/g1/contests/c1/standings",
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var raw struct {
		Standings []map[string]any `json:"standings"`
	}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Standings) != 2 {
		t.Fatalf("standings = %d rows, want 2 (official then post)", len(raw.Standings))
	}
	if raw.Standings[0]["isOfficial"] != true || raw.Standings[0]["rank"] != float64(1) {
		t.Errorf("first row = %v, want official with numeric rank 1", raw.Standings[0])
	}
	if raw.Standings[1]["isOfficial"] != false {
		t.Errorf("second row isOfficial = %v, want false", raw.Standings[1]["isOfficial"])
	}
	if rank, present := raw.Standings[1]["rank"]; !present || rank != nil {
		t.Errorf("post row rank = %v (present=%v), want explicit null", rank, present)
	}
}

func TestGetStandings_PostRow_HasNullTimeZeroPenaltyAndNoFirstSolve(t *testing.T) {
	late := time.Now().Add(-30 * time.Minute)
	w := serveStandings(t,
		postStandingsUC(nil,
			[]domainContest.ParticipantStanding{{ContestantID: "p1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &late}}}},
		),
		"/groups/g1/contests/c1/standings?phase=postcompetition",
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var raw struct {
		Standings []struct {
			IsOfficial bool             `json:"isOfficial"`
			Problems   []map[string]any `json:"problems"`
		} `json:"standings"`
	}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Standings) != 1 || raw.Standings[0].IsOfficial {
		t.Fatalf("standings = %+v, want one unofficial row", raw.Standings)
	}
	cell := raw.Standings[0].Problems[0]
	if cell["status"] != "ACCEPTED" || cell["time"] != nil || cell["penalty"] != float64(0) || cell["isFirstSolve"] != false {
		t.Errorf("post cell = %v, want ACCEPTED with time null, penalty 0, isFirstSolve false", cell)
	}
}

func TestGetStandings_PhaseCompetition_ReturnsOnlyOfficialRows(t *testing.T) {
	early := time.Now().Add(-2 * time.Hour)
	late := time.Now().Add(-30 * time.Minute)
	w := serveStandings(t,
		postStandingsUC(
			[]domainContest.ParticipantStanding{{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &early}}}},
			[]domainContest.ParticipantStanding{{ContestantID: "p1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &late}}}},
		),
		"/groups/g1/contests/c1/standings?phase=competition",
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Standings) != 1 || resp.Standings[0].Participant.ID != "u1" {
		t.Errorf("standings = %+v, want only u1", resp.Standings)
	}
}

// ── pending cells ─────────────────────────────────────────────────────────────

func frozenBoardStandingsUC(participants ...domainContest.ParticipantStanding) *appcontest.GetStandingsUseCase {
	contest := domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Frozen Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(30*time.Minute),
		domainContest.RestorePenalty(20),
		60, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{
			domainContest.RestoreContestProblem("cp1", "prob-A", 1, nil),
			domainContest.RestoreContestProblem("cp2", "prob-B", 2, nil),
		},
		time.Now(),
		nil,
	)
	cached := &appcontest.CachedStandings{Participants: participants, LastUpdated: time.Now()}
	return appcontest.NewGetStandingsUseCase(
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
}

func firstStandingCells(t *testing.T, w *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Standings []struct {
			Problems []map[string]any `json:"problems"`
		} `json:"standings"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Standings) == 0 {
		t.Fatal("no standings")
	}
	return body.Standings[0].Problems
}

func TestGetStandings_FrozenCell_IsPendingWithTheCount(t *testing.T) {
	now := time.Now()
	accepted := now.Add(-10 * time.Minute)
	uc := frozenBoardStandingsUC(domainContest.ParticipantStanding{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
		"prob-A": {WrongAttemptTimes: []time.Time{now.Add(-100 * time.Minute), now.Add(-20 * time.Minute)}, AcceptedAt: &accepted},
	}})

	cell := firstStandingCells(t, serveStandings(t, uc, "/groups/g1/contests/c1/standings"))[0]

	want := map[string]any{"status": "PENDING", "attempts": float64(1), "pending": float64(2), "time": nil, "penalty": float64(0), "isFirstSolve": false}
	for k, v := range want {
		if cell[k] != v {
			t.Errorf("%s = %v, want %v", k, cell[k], v)
		}
	}
}

func TestGetStandings_PendingKeyIsAlwaysPresent(t *testing.T) {
	accepted := time.Now().Add(-60 * time.Minute)
	uc := frozenBoardStandingsUC(domainContest.ParticipantStanding{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
		"prob-A": {AcceptedAt: &accepted},
	}})

	cells := firstStandingCells(t, serveStandings(t, uc, "/groups/g1/contests/c1/standings"))

	if cells[0]["status"] != "ACCEPTED" {
		t.Errorf("prob-A status = %v, want ACCEPTED", cells[0]["status"])
	}
	if cells[1]["status"] != "NOT_ATTEMPTED" {
		t.Errorf("prob-B status = %v, want NOT_ATTEMPTED", cells[1]["status"])
	}
	for i, cell := range cells {
		if v, present := cell["pending"]; !present || v != float64(0) {
			t.Errorf("cell %d pending = %v (present=%v), want an explicit 0", i, v, present)
		}
	}
}

func TestGetStandings_RealtimeLead_HasNoPendingCells(t *testing.T) {
	now := time.Now()
	accepted := now.Add(-10 * time.Minute)
	uc := frozenBoardStandingsUC(domainContest.ParticipantStanding{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
		"prob-A": {WrongAttemptTimes: []time.Time{now.Add(-100 * time.Minute), now.Add(-20 * time.Minute)}, AcceptedAt: &accepted},
	}})

	cell := firstStandingCells(t, serveStandings(t, uc, "/groups/g1/contests/c1/standings?realtime=true"))[0]

	if cell["status"] != "ACCEPTED" {
		t.Errorf("status = %v, want ACCEPTED in real time", cell["status"])
	}
	if cell["time"] == nil {
		t.Error("time is null, want the official solve minute in real time")
	}
	if cell["pending"] != float64(0) {
		t.Errorf("pending = %v, want 0 in real time", cell["pending"])
	}
}

func TestGetStandings_SolvedBeforeTheFreeze_StaysAccepted(t *testing.T) {
	now := time.Now()
	accepted := now.Add(-60 * time.Minute)
	uc := frozenBoardStandingsUC(domainContest.ParticipantStanding{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
		"prob-A": {WrongAttemptTimes: []time.Time{now.Add(-20 * time.Minute)}, AcceptedAt: &accepted},
	}})

	cell := firstStandingCells(t, serveStandings(t, uc, "/groups/g1/contests/c1/standings"))[0]

	if cell["status"] != "ACCEPTED" {
		t.Errorf("status = %v, want ACCEPTED", cell["status"])
	}
	if cell["pending"] != float64(0) {
		t.Errorf("pending = %v, want 0", cell["pending"])
	}
}

func decodeStandingsHeaders(t *testing.T, w *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var raw struct {
		Problems []map[string]any `json:"problems"`
	}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return raw.Problems
}

func TestGetStandings_Headers_CarryThePostCountersApartFromTheOfficialOnes(t *testing.T) {
	early := time.Now().Add(-2 * time.Hour)
	late := time.Now().Add(-30 * time.Minute)
	w := serveStandings(t,
		postStandingsUC(
			[]domainContest.ParticipantStanding{{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &early}}}},
			[]domainContest.ParticipantStanding{
				{ContestantID: "p1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &late}}},
				{ContestantID: "p2", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {WrongAttemptTimes: []time.Time{late}}}},
			},
		),
		"/groups/g1/contests/c1/standings?phase=all",
	)

	headers := decodeStandingsHeaders(t, w)

	if len(headers) != 1 {
		t.Fatalf("headers = %v, want one problem", headers)
	}
	h := headers[0]
	if h["accepted"] != float64(1) || h["tried"] != float64(1) {
		t.Errorf("accepted/tried = %v/%v, want the official 1/1", h["accepted"], h["tried"])
	}
	if h["postAccepted"] != float64(1) || h["postTried"] != float64(2) {
		t.Errorf("postAccepted/postTried = %v/%v, want 1/2", h["postAccepted"], h["postTried"])
	}
}

func TestGetStandings_Headers_PostCountersAreAlwaysPresent(t *testing.T) {
	early := time.Now().Add(-2 * time.Hour)
	w := serveStandings(t,
		postStandingsUC(
			[]domainContest.ParticipantStanding{{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{"prob-A": {AcceptedAt: &early}}}},
			nil,
		),
		"/groups/g1/contests/c1/standings?phase=competition",
	)

	headers := decodeStandingsHeaders(t, w)

	for _, key := range []string{"postAccepted", "postTried"} {
		if v, present := headers[0][key]; !present || v != float64(0) {
			t.Errorf("%s = %v (present=%v), want an explicit 0", key, v, present)
		}
	}
}
