package contest

import (
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func standingOf(contestantID, participantType string) domainContest.ParticipantStanding {
	return domainContest.ParticipantStanding{
		ContestantID:    contestantID,
		ParticipantType: participantType,
		Problems:        map[string]domainContest.ProblemAttempt{},
	}
}

func TestFilterStandingsByProfile_EmptyFiltersReturnsAllUnchanged(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			standingOf("u1", "INDIVIDUAL"),
			standingOf("u2", "INDIVIDUAL"),
		},
		Profiles: nil, // never touched when filters are empty
	}

	out := FilterStandingsByProfile(cached, "", "", "")

	if len(out) != 2 {
		t.Fatalf("expected all 2 participants unchanged, got %d", len(out))
	}
}

func TestFilterStandingsByProfile_AndCombination(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			standingOf("u1", "INDIVIDUAL"), // matches both
			standingOf("u2", "INDIVIDUAL"), // matches country only
		},
		Profiles: map[string]*ParticipantProfile{
			"u1": {ID: "u1", Country: "colombia", City: "bogota"},
			"u2": {ID: "u2", Country: "colombia", City: "medellin"},
		},
	}

	out := FilterStandingsByProfile(cached, "colombia", "bogota", "")

	if len(out) != 1 || out[0].ContestantID != "u1" {
		t.Fatalf("expected only u1 to match country AND city, got %+v", out)
	}
}

func TestFilterStandingsByProfile_CaseInsensitive(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{standingOf("u1", "INDIVIDUAL")},
		Profiles: map[string]*ParticipantProfile{
			"u1": {ID: "u1", Country: "colombia"},
		},
	}

	out := FilterStandingsByProfile(cached, "Colombia", "", "")

	if len(out) != 1 {
		t.Fatalf("expected case-insensitive match, got %d", len(out))
	}
}

func TestFilterStandingsByProfile_TeamPassesIfAnyMemberMatches(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{standingOf("team-1", "TEAM")},
		TeamMembers: map[string][]string{
			"team-1": {"m1", "m2", "m3"},
		},
		Profiles: map[string]*ParticipantProfile{
			"m1": {ID: "m1", Country: "mexico"},
			"m2": {ID: "m2", Country: "colombia"}, // the only one that matches
			"m3": {ID: "m3", Country: "peru"},
		},
	}

	out := FilterStandingsByProfile(cached, "colombia", "", "")

	if len(out) != 1 || out[0].ContestantID != "team-1" {
		t.Fatalf("expected the team to pass because one member matches, got %+v", out)
	}
}

func TestFilterStandingsByProfile_TeamExcludedWhenNoMemberMatches(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{standingOf("team-1", "TEAM")},
		TeamMembers: map[string][]string{
			"team-1": {"m1", "m2"},
		},
		Profiles: map[string]*ParticipantProfile{
			"m1": {ID: "m1", Country: "mexico"},
			"m2": {ID: "m2", Country: "peru"},
		},
	}

	out := FilterStandingsByProfile(cached, "colombia", "", "")

	if len(out) != 0 {
		t.Fatalf("expected the team to be excluded, got %+v", out)
	}
}

func TestFilterStandingsByProfile_MissingProfileDoesNotMatch(t *testing.T) {
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{standingOf("u1", "INDIVIDUAL")},
		Profiles:     map[string]*ParticipantProfile{}, // u1 has no entry
	}

	out := FilterStandingsByProfile(cached, "colombia", "", "")

	if len(out) != 0 {
		t.Fatalf("expected participant with no profile to be excluded, not panic, got %+v", out)
	}
}

func TestFilterStandingsByProfile_RanksRecomputedRelativeToFilteredSubset(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	accepted := func(minutesAfterStart int) *time.Time {
		t := start.Add(time.Duration(minutesAfterStart) * time.Minute)
		return &t
	}

	// 5 participants overall; only 2 are from "colombia".
	// Full-field rank order (by ProblemsSolved desc): u1(2) > u2(1) > u3(1) > u4(0) > u5(0)
	// So u3 is rank 3 overall. After filtering to colombia (u3, u5 only),
	// u3 should become rank 1 of 2.
	cached := &CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
				"A": {AcceptedAt: accepted(10)}, "B": {AcceptedAt: accepted(20)},
			}},
			{ContestantID: "u2", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
				"A": {AcceptedAt: accepted(15)},
			}},
			{ContestantID: "u3", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{
				"A": {AcceptedAt: accepted(25)},
			}},
			{ContestantID: "u4", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u5", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{}},
		},
		Profiles: map[string]*ParticipantProfile{
			"u1": {ID: "u1", Country: "mexico"},
			"u2": {ID: "u2", Country: "peru"},
			"u3": {ID: "u3", Country: "colombia"},
			"u4": {ID: "u4", Country: "mexico"},
			"u5": {ID: "u5", Country: "colombia"},
		},
	}

	// Sanity check: u3 is rank 3 overall, unfiltered.
	overall := RankStandings(FilterStandingsByProfile(cached, "", "", ""), start, 20, nil)
	var u3RankOverall int
	for _, e := range overall {
		if e.ContestantID == "u3" {
			u3RankOverall = e.Rank
		}
	}
	if u3RankOverall != 3 {
		t.Fatalf("expected u3 to be rank 3 overall, got %d", u3RankOverall)
	}

	filtered := RankStandings(FilterStandingsByProfile(cached, "colombia", "", ""), start, 20, nil)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 colombia participants, got %d", len(filtered))
	}
	if filtered[0].ContestantID != "u3" || filtered[0].Rank != 1 {
		t.Fatalf("expected u3 to be rank 1 of the filtered subset, got %+v", filtered[0])
	}
}
