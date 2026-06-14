package contest_test

import (
	"testing"
	"time"

	appContest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

var (
	contestStart   = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	contestPenalty = 20 // minutes per wrong attempt
)

func tp(h, m int) *time.Time {
	t := time.Date(2026, 1, 1, h, m, 0, 0, time.UTC)
	return &t
}

func participant(id string, problems map[string]domainContest.ProblemAttempt) domainContest.ParticipantStanding {
	return domainContest.ParticipantStanding{ContestantID: id, Problems: problems}
}

func attempt(wrongAttempts int, acceptedAt *time.Time) domainContest.ProblemAttempt {
	return domainContest.ProblemAttempt{Attempts: wrongAttempts, AcceptedAt: acceptedAt}
}

func TestRankStandings(t *testing.T) {
	tests := []struct {
		name         string
		participants []domainContest.ParticipantStanding
		freezeTime   *time.Time
		wantOrder    []string // contestantIDs in expected rank order
		wantRanks    []int
		wantSolved   []int
		wantPenalty  []int
	}{
		{
			name: "single participant no problems",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{}),
			},
			wantOrder:   []string{"u1"},
			wantRanks:   []int{1},
			wantSolved:  []int{0},
			wantPenalty: []int{0},
		},
		{
			name: "two participants sorted by problems solved",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)), // solved A at 10:30 → 30 min, penalty 30
				}),
				participant("u2", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 20)), // solved A at 10:20 → 20 min
					"B": attempt(0, tp(10, 50)), // solved B at 10:50 → 50 min
				}),
			},
			wantOrder:   []string{"u2", "u1"},
			wantRanks:   []int{1, 2},
			wantSolved:  []int{2, 1},
			wantPenalty: []int{70, 30},
		},
		{
			name: "tie broken by total penalty",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 60)), // 60 min
				}),
				participant("u2", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)), // 30 min — lower penalty wins
				}),
			},
			wantOrder:   []string{"u2", "u1"},
			wantRanks:   []int{1, 2},
			wantSolved:  []int{1, 1},
			wantPenalty: []int{30, 60},
		},
		{
			name: "tie broken by last accepted time",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)), // 30 min penalty, last=10:30
				}),
				participant("u2", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 20)), // 20 min, BUT add wrong attempt to equalize
					// wrong attempt adds 20 → total 20+20=40? No, want same penalty.
					// Let's do: u2 solved A with 1 wrong attempt at 10:10 → 10 + 20 = 30. Same penalty.
				}),
			},
			// u1: A solved at 10:30, 0 wrong, penalty=30, last=10:30
			// u2: A solved at 10:10 (not 10:20), 1 wrong, penalty=10+20=30, last=10:10
			// same solved (1), same penalty (30), u2 last accepted earlier → u2 wins
			wantOrder:   []string{"u1", "u2"}, // will redefine below
			wantRanks:   []int{1, 2},
			wantSolved:  []int{1, 1},
			wantPenalty: []int{30, 30},
		},
		{
			name: "wrong attempts add penalty",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(2, tp(11, 0)), // 60 min + 2*20 = 100 penalty
				}),
			},
			wantOrder:   []string{"u1"},
			wantRanks:   []int{1},
			wantSolved:  []int{1},
			wantPenalty: []int{100},
		},
		{
			name: "unsolved problem does not add penalty",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)), // solved
					"B": attempt(5, nil),         // not solved — 5 wrong attempts, no penalty
				}),
			},
			wantOrder:   []string{"u1"},
			wantRanks:   []int{1},
			wantSolved:  []int{1},
			wantPenalty: []int{30},
		},
		{
			name: "freeze hides accepted after freeze time",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)), // before freeze → counts
					"B": attempt(0, tp(11, 30)), // after freeze → hidden
				}),
			},
			freezeTime:  tp(11, 0),
			wantOrder:   []string{"u1"},
			wantRanks:   []int{1},
			wantSolved:  []int{1},
			wantPenalty: []int{30},
		},
		{
			name: "freeze does not affect finished contest (nil freeze)",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(11, 30)), // would be after a freeze, but freeze is nil
				}),
			},
			freezeTime:  nil,
			wantOrder:   []string{"u1"},
			wantRanks:   []int{1},
			wantSolved:  []int{1},
			wantPenalty: []int{90},
		},
		{
			name: "exact tie gets same rank",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)),
				}),
				participant("u2", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)),
				}),
			},
			wantOrder:   []string{"u1", "u2"}, // order between tied is stable but both rank 1
			wantRanks:   []int{1, 1},
			wantSolved:  []int{1, 1},
			wantPenalty: []int{30, 30},
		},
		{
			name: "participant with no submissions appears last with zeros",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{
					"A": attempt(0, tp(10, 30)),
				}),
				participant("u2", map[string]domainContest.ProblemAttempt{}),
			},
			wantOrder:   []string{"u1", "u2"},
			wantRanks:   []int{1, 2},
			wantSolved:  []int{1, 0},
			wantPenalty: []int{30, 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Patch the last-accepted-tie test case properly
			if tc.name == "tie broken by last accepted time" {
				tc.participants = []domainContest.ParticipantStanding{
					participant("u1", map[string]domainContest.ProblemAttempt{
						"A": attempt(0, tp(10, 30)), // penalty=30, last=10:30
					}),
					participant("u2", map[string]domainContest.ProblemAttempt{
						"A": attempt(1, tp(10, 10)), // penalty=10+20=30, last=10:10 — earlier wins
					}),
				}
				tc.wantOrder = []string{"u2", "u1"}
			}

			got := appContest.RankStandings(tc.participants, contestStart, contestPenalty, tc.freezeTime)

			if len(got) != len(tc.wantOrder) {
				t.Fatalf("len(got)=%d, want %d", len(got), len(tc.wantOrder))
			}

			for i, entry := range got {
				if entry.ContestantID != tc.wantOrder[i] {
					t.Errorf("[%d] contestantID=%q, want %q", i, entry.ContestantID, tc.wantOrder[i])
				}
				if entry.Rank != tc.wantRanks[i] {
					t.Errorf("[%d] rank=%d, want %d", i, entry.Rank, tc.wantRanks[i])
				}
				if entry.ProblemsSolved != tc.wantSolved[i] {
					t.Errorf("[%d] solved=%d, want %d", i, entry.ProblemsSolved, tc.wantSolved[i])
				}
				if entry.TotalPenalty != tc.wantPenalty[i] {
					t.Errorf("[%d] penalty=%d, want %d", i, entry.TotalPenalty, tc.wantPenalty[i])
				}
			}
		})
	}
}
