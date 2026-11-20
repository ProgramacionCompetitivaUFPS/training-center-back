package contest

import (
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func wrongAt(h, m int) []time.Time { return []time.Time{*tp(h, m)} }

func TestComputeProblemStats(t *testing.T) {
	freeze := tp(12, 0)

	tests := []struct {
		name         string
		participants []domainContest.ParticipantStanding
		freezeTime   *time.Time
		problem      string
		want         problemStats
	}{
		{
			name:    "no participants",
			problem: "A",
			want:    problemStats{},
		},
		{
			name: "a problem nobody touched has no stats",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"B": attempt(0, tp(10, 30))}),
			},
			problem: "A",
			want:    problemStats{},
		},
		{
			name: "solved counts as accepted and tried, failed only as tried",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(2, nil)}),
				participant("u3", map[string]domainContest.ProblemAttempt{}),
			},
			problem: "A",
			want:    problemStats{Accepted: 1, Tried: 2, FirstSolverID: "u1"},
		},
		{
			name: "a solve after failed attempts counts once",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(3, tp(10, 40))}),
			},
			problem: "A",
			want:    problemStats{Accepted: 1, Tried: 1, FirstSolverID: "u1"},
		},
		{
			name: "first solver is the earliest acceptance, not the first listed",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 50))}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 20))}),
				participant("u3", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 35))}),
			},
			problem: "A",
			want:    problemStats{Accepted: 3, Tried: 3, FirstSolverID: "u2"},
		},
		{
			name: "a tie on the acceptance time goes to the smaller contestant id",
			participants: []domainContest.ParticipantStanding{
				participant("u9", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
			},
			problem: "A",
			want:    problemStats{Accepted: 2, Tried: 2, FirstSolverID: "u2"},
		},
		{
			name: "a tie on the acceptance time goes to the smaller id whatever the order",
			participants: []domainContest.ParticipantStanding{
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
				participant("u9", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
			},
			problem: "A",
			want:    problemStats{Accepted: 2, Tried: 2, FirstSolverID: "u2"},
		},
		{
			name: "an earlier failed attempt of someone else does not take the first solve",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": {WrongAttemptTimes: wrongAt(10, 0)}}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
			},
			problem: "A",
			want:    problemStats{Accepted: 1, Tried: 2, FirstSolverID: "u2"},
		},
		{
			name: "a solve after the freeze is hidden",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30))}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(12, 30))}),
			},
			freezeTime: freeze,
			problem:    "A",
			want:       problemStats{Accepted: 1, Tried: 1, FirstSolverID: "u1"},
		},
		{
			name: "attempts after the freeze are hidden",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": {WrongAttemptTimes: wrongAt(12, 30)}}),
				participant("u2", map[string]domainContest.ProblemAttempt{"A": {WrongAttemptTimes: wrongAt(11, 30)}}),
			},
			freezeTime: freeze,
			problem:    "A",
			want:       problemStats{Tried: 1},
		},
		{
			name: "a failed attempt before the freeze still counts when the solve is hidden",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": {WrongAttemptTimes: wrongAt(11, 0), AcceptedAt: tp(12, 30)}}),
			},
			freezeTime: freeze,
			problem:    "A",
			want:       problemStats{Tried: 1},
		},
		{
			name: "a solve exactly at the freeze is still visible",
			participants: []domainContest.ParticipantStanding{
				participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(12, 0))}),
			},
			freezeTime: freeze,
			problem:    "A",
			want:       problemStats{Accepted: 1, Tried: 1, FirstSolverID: "u1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeProblemStats(tt.participants, tt.freezeTime)[tt.problem]

			if got != tt.want {
				t.Errorf("stats for %s = %+v, want %+v", tt.problem, got, tt.want)
			}
		})
	}
}

func TestComputeProblemStats_ProblemsAreIndependent(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		participant("u1", map[string]domainContest.ProblemAttempt{"A": attempt(0, tp(10, 30)), "B": attempt(1, nil)}),
		participant("u2", map[string]domainContest.ProblemAttempt{"B": attempt(0, tp(10, 20))}),
	}

	got := computeProblemStats(participants, nil)

	if want := (problemStats{Accepted: 1, Tried: 1, FirstSolverID: "u1"}); got["A"] != want {
		t.Errorf("A = %+v, want %+v", got["A"], want)
	}
	if want := (problemStats{Accepted: 1, Tried: 2, FirstSolverID: "u2"}); got["B"] != want {
		t.Errorf("B = %+v, want %+v", got["B"], want)
	}
}

func TestPendingSubmissions(t *testing.T) {
	freeze := tp(12, 0)

	tests := []struct {
		name     string
		attempt  domainContest.ProblemAttempt
		noFreeze bool
		want     int
	}{
		{
			name:     "no freeze counts nothing",
			attempt:  domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(13, 0), *tp(13, 5)}, AcceptedAt: tp(13, 10)},
			noFreeze: true,
			want:     0,
		},
		{
			name:    "only failures before the freeze",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(10, 30), *tp(11, 45)}},
			want:    0,
		},
		{
			name:    "two failures after the freeze",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 10), *tp(12, 20)}},
			want:    2,
		},
		{
			name:    "only an acceptance after the freeze",
			attempt: domainContest.ProblemAttempt{AcceptedAt: tp(12, 30)},
			want:    1,
		},
		{
			name:    "one failure and an acceptance after the freeze",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 10)}, AcceptedAt: tp(12, 20)},
			want:    2,
		},
		{
			name: "failures before the freeze do not count",
			attempt: domainContest.ProblemAttempt{
				WrongAttemptTimes: []time.Time{*tp(10, 30), *tp(10, 45), *tp(12, 10)},
				AcceptedAt:        tp(12, 20),
			},
			want: 2,
		},
		{
			name:    "solved before the freeze counts nothing even with later failures",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 10)}, AcceptedAt: tp(10, 30)},
			want:    0,
		},
		{
			name:    "acceptance exactly at the freeze is still visible",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 10)}, AcceptedAt: tp(12, 0)},
			want:    0,
		},
		{
			name:    "a failure exactly at the freeze is still visible",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 0)}},
			want:    0,
		},
		{
			name:    "a failure exactly at the freeze stays visible while a later one is pending",
			attempt: domainContest.ProblemAttempt{WrongAttemptTimes: []time.Time{*tp(12, 0), *tp(12, 10)}},
			want:    1,
		},
		{
			name:    "empty attempt counts nothing",
			attempt: domainContest.ProblemAttempt{},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := freeze
			if tt.noFreeze {
				f = nil
			}
			if got := pendingSubmissions(tt.attempt, f); got != tt.want {
				t.Errorf("pendingSubmissions() = %d, want %d", got, tt.want)
			}
		})
	}
}
