package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

const (
	boardProblemA = "A"
	boardProblemB = "B"
)

func boardContest(base *domainContest.Contest) *domainContest.Contest {
	base.SetProblems([]domainContest.ContestProblem{
		problemWithColor("cp-a", boardProblemA, 1, "#aabbcc"),
		domainContest.RestoreContestProblem("cp-b", boardProblemB, 2, nil),
	}, testNow)
	return base
}

func solvedAt(ago time.Duration) domainContest.ProblemAttempt {
	t := time.Now().Add(-ago)
	return domainContest.ProblemAttempt{AcceptedAt: &t}
}

func boardParticipant(id string, problems map[string]domainContest.ProblemAttempt) domainContest.ParticipantStanding {
	return domainContest.ParticipantStanding{ContestantID: id, ParticipantType: "INDIVIDUAL", Problems: problems}
}

func boardCache(profiles map[string]*ParticipantProfile, participants ...domainContest.ParticipantStanding) *mockStandingsCache {
	cached := &CachedStandings{Participants: participants, Profiles: profiles, LastUpdated: time.Now()}
	return &mockStandingsCache{
		getFn: func(_ context.Context, _ string) (*CachedStandings, error) { return cached, nil },
	}
}

func executeBoard(t *testing.T, contest *domainContest.Contest, cache *mockStandingsCache, mutate func(*GetStandingsInput)) *GetStandingsOutput {
	t.Helper()
	uc := newGetStandingsUseCase(contest, nil, nil, cache, visibleGroup(), isMemberNotLead())
	in := GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	}
	if mutate != nil {
		mutate(&in)
	}
	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

func entryFor(t *testing.T, out *GetStandingsOutput, contestantID string) RankedEntry {
	t.Helper()
	for _, e := range out.Entries {
		if e.ContestantID == contestantID {
			return e
		}
	}
	t.Fatalf("no entry for %s in %+v", contestantID, out.Entries)
	return RankedEntry{}
}

func headerFor(t *testing.T, out *GetStandingsOutput, problemID string) StandingsProblemDisplay {
	t.Helper()
	for _, p := range out.Problems {
		if p.ID == problemID {
			return p
		}
	}
	t.Fatalf("no header for %s in %+v", problemID, out.Problems)
	return StandingsProblemDisplay{}
}

func TestGetStandingsUseCase_ProblemHeadersCarryColorAndCounts(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(90 * time.Minute)}),
		boardParticipant("u2", map[string]domainContest.ProblemAttempt{boardProblemA: {WrongAttemptTimes: []time.Time{time.Now().Add(-80 * time.Minute)}}}),
		boardParticipant("u3", map[string]domainContest.ProblemAttempt{}),
	)

	out := executeBoard(t, boardContest(activeContest()), cache, nil)

	a, b := headerFor(t, out, boardProblemA), headerFor(t, out, boardProblemB)
	assertColor(t, "problem A", a.Color, "#aabbcc")
	assertNoColor(t, "problem B", b.Color)
	if a.Accepted != 1 || a.Tried != 2 {
		t.Errorf("problem A accepted/tried = %d/%d, want 1/2", a.Accepted, a.Tried)
	}
	if b.Accepted != 0 || b.Tried != 0 {
		t.Errorf("problem B accepted/tried = %d/%d, want 0/0", b.Accepted, b.Tried)
	}
}

func TestGetStandingsUseCase_OnlyTheEarliestSolveIsMarkedFirst(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(60 * time.Minute)}),
		boardParticipant("u2", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(100 * time.Minute), boardProblemB: solvedAt(70 * time.Minute)}),
	)

	out := executeBoard(t, boardContest(activeContest()), cache, nil)

	if !entryFor(t, out, "u2").Problems[boardProblemA].IsFirstSolve {
		t.Error("u2 solved A first and must be marked")
	}
	if entryFor(t, out, "u1").Problems[boardProblemA].IsFirstSolve {
		t.Error("u1 solved A later and must not be marked")
	}
	if !entryFor(t, out, "u2").Problems[boardProblemB].IsFirstSolve {
		t.Error("each problem has its own first solve: u2 is also first on B")
	}
}

func TestGetStandingsUseCase_AFailedAttemptIsNeverAFirstSolve(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: {WrongAttemptTimes: []time.Time{time.Now().Add(-80 * time.Minute)}}}),
	)

	out := executeBoard(t, boardContest(activeContest()), cache, nil)

	if entryFor(t, out, "u1").Problems[boardProblemA].IsFirstSolve {
		t.Error("a problem nobody solved has no first solve")
	}
}

func TestGetStandingsUseCase_CountsAndFirstSolveIgnoreTheProfileFilter(t *testing.T) {
	profiles := map[string]*ParticipantProfile{
		"u1": {ID: "u1", Country: "Colombia"},
		"u2": {ID: "u2", Country: "Peru"},
	}
	cache := boardCache(profiles,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(60 * time.Minute)}),
		boardParticipant("u2", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(100 * time.Minute)}),
	)

	out := executeBoard(t, boardContest(activeContest()), cache, func(in *GetStandingsInput) { in.Country = "Colombia" })

	if len(out.Entries) != 1 || out.Entries[0].ContestantID != "u1" {
		t.Fatalf("entries = %+v, want only u1", out.Entries)
	}
	if a := headerFor(t, out, boardProblemA); a.Accepted != 2 || a.Tried != 2 {
		t.Errorf("problem A accepted/tried = %d/%d, want 2/2 over every participant", a.Accepted, a.Tried)
	}
	if out.Entries[0].Problems[boardProblemA].IsFirstSolve {
		t.Error("the first solver is outside the filter: u1 must not be marked")
	}
}

func TestGetStandingsUseCase_FrozenBoardHidesCountsAndFirstSolveAfterTheFreeze(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(20 * time.Minute)}),
		boardParticipant("u2", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(100 * time.Minute)}),
	)

	out := executeBoard(t, boardContest(frozenContest()), cache, nil)

	if a := headerFor(t, out, boardProblemA); a.Accepted != 1 || a.Tried != 1 {
		t.Errorf("problem A accepted/tried = %d/%d, want 1/1 (the solve after the freeze is hidden)", a.Accepted, a.Tried)
	}
	if entryFor(t, out, "u1").Problems[boardProblemA].IsFirstSolve {
		t.Error("a hidden solve cannot be a first solve")
	}
	if !entryFor(t, out, "u2").Problems[boardProblemA].IsFirstSolve {
		t.Error("u2 solved before the freeze and is the first visible solve")
	}
}

func TestGetStandingsUseCase_RealtimeShowsTheSolvesHiddenByTheFreeze(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(20 * time.Minute)}),
	)
	uc := newGetStandingsUseCase(boardContest(frozenContest()), nil, nil, cache, visibleGroup(), isLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Realtime:    true,
		Page:        1,
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a := headerFor(t, out, boardProblemA); a.Accepted != 1 {
		t.Errorf("problem A accepted = %d, want 1 in real time", a.Accepted)
	}
	if !entryFor(t, out, "u1").Problems[boardProblemA].IsFirstSolve {
		t.Error("in real time the post-freeze solve is the first solve")
	}
}
