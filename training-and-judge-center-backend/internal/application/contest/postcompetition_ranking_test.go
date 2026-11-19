package contest

import (
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func rankedIDs(entries []RankedEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ContestantID
	}
	return ids
}

func nameFrom(names map[string]string) func(string, string) string {
	return func(id, _ string) string { return names[id] }
}

func TestRankPostcompetition_OrdersByProblemsSolvedDesc(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("a", map[string]domainContest.ProblemAttempt{"p1": attempt(0, tp(13, 1))}),
		participant("b", map[string]domainContest.ProblemAttempt{
			"p1": attempt(0, tp(13, 5)),
			"p2": attempt(0, tp(13, 6)),
		}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "A", "b": "B"}))
	if want := []string{"b", "a"}; !equalStrings(rankedIDs(got), want) {
		t.Errorf("order = %v, want %v", rankedIDs(got), want)
	}
}

func TestRankPostcompetition_TieBrokenByEarliestLastAccept(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("a", map[string]domainContest.ProblemAttempt{"p1": attempt(0, tp(13, 5))}),
		participant("b", map[string]domainContest.ProblemAttempt{"p1": attempt(0, tp(13, 2))}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "A", "b": "B"}))
	if want := []string{"b", "a"}; !equalStrings(rankedIDs(got), want) {
		t.Errorf("order = %v, want %v", rankedIDs(got), want)
	}
}

func TestRankPostcompetition_ZeroSolvedRowsLastOrderedByName(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("zed", map[string]domainContest.ProblemAttempt{"p1": attempt(2, nil)}),
		participant("bob", map[string]domainContest.ProblemAttempt{"p1": attempt(1, tp(13, 1))}),
		participant("ann", map[string]domainContest.ProblemAttempt{"p1": attempt(3, nil)}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"zed": "Zed", "bob": "Bob", "ann": "Ann"}))
	if want := []string{"bob", "ann", "zed"}; !equalStrings(rankedIDs(got), want) {
		t.Errorf("order = %v, want %v", rankedIDs(got), want)
	}
	if got[1].LastAcceptedAt != nil {
		t.Errorf("zero-solved row LastAcceptedAt = %v, want nil", got[1].LastAcceptedAt)
	}
}

func TestRankPostcompetition_NameTiesOrderedByContestantID(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("b", map[string]domainContest.ProblemAttempt{"p1": attempt(1, nil)}),
		participant("a", map[string]domainContest.ProblemAttempt{"p1": attempt(1, nil)}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "Same", "b": "Same"}))
	if want := []string{"a", "b"}; !equalStrings(rankedIDs(got), want) {
		t.Errorf("order = %v, want %v", rankedIDs(got), want)
	}
}

func TestRankPostcompetition_NameComparisonIgnoresCase(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("b", map[string]domainContest.ProblemAttempt{"p1": attempt(1, nil)}),
		participant("a", map[string]domainContest.ProblemAttempt{"p1": attempt(1, nil)}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "Banana", "b": "apple"}))
	if want := []string{"b", "a"}; !equalStrings(rankedIDs(got), want) {
		t.Errorf("order = %v, want %v", rankedIDs(got), want)
	}
}

func TestRankPostcompetition_AttemptsExcludeTheAccept(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("a", map[string]domainContest.ProblemAttempt{
			"p1": attempt(2, tp(13, 5)),
			"p2": attempt(0, tp(13, 6)),
		}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "A"}))
	if n := got[0].Problems["p1"].Attempts; n != 2 {
		t.Errorf("p1 attempts = %d, want 2", n)
	}
	if n := got[0].Problems["p2"].Attempts; n != 0 {
		t.Errorf("p2 attempts = %d, want 0", n)
	}
}

func TestRankPostcompetition_RowsAreUnofficialWithoutRankOrPenalty(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("a", map[string]domainContest.ProblemAttempt{"p1": attempt(3, tp(13, 5))}),
	}
	got := RankPostcompetition(participants, nameFrom(map[string]string{"a": "A"}))
	e := got[0]
	if e.IsOfficial {
		t.Error("IsOfficial = true, want false")
	}
	if e.Rank != 0 {
		t.Errorf("Rank = %d, want 0", e.Rank)
	}
	if e.TotalPenalty != 0 || e.Problems["p1"].Penalty != 0 {
		t.Errorf("penalty = %d/%d, want 0", e.TotalPenalty, e.Problems["p1"].Penalty)
	}
	if e.ProblemsSolved != 1 {
		t.Errorf("ProblemsSolved = %d, want 1", e.ProblemsSolved)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
