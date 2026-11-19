package contest

import (
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

// problemStats are the board totals of one problem over every participant.
type problemStats struct {
	Accepted      int
	Tried         int
	FirstSolverID string
}

// visibleAttempt returns the wrong attempts and whether the solve is visible at freezeTime (nil = no freeze).
func visibleAttempt(attempt domainContest.ProblemAttempt, freezeTime *time.Time) (wrong int, solved bool) {
	for _, t := range attempt.WrongAttemptTimes {
		if freezeTime == nil || !t.After(*freezeTime) {
			wrong++
		}
	}
	solved = attempt.AcceptedAt != nil && (freezeTime == nil || !attempt.AcceptedAt.After(*freezeTime))
	return wrong, solved
}

// computeProblemStats counts per problem who tried, who solved and who solved first (a time tie goes to the smaller ID).
func computeProblemStats(participants []domainContest.ParticipantStanding, freezeTime *time.Time) map[string]problemStats {
	stats := make(map[string]problemStats)
	firstAt := make(map[string]time.Time)

	for _, p := range participants {
		for problemID, attempt := range p.Problems {
			wrong, solved := visibleAttempt(attempt, freezeTime)
			if wrong == 0 && !solved {
				continue
			}
			s := stats[problemID]
			s.Tried++
			if solved {
				s.Accepted++
				at := *attempt.AcceptedAt
				prev, seen := firstAt[problemID]
				if !seen || at.Before(prev) || (at.Equal(prev) && p.ContestantID < s.FirstSolverID) {
					firstAt[problemID] = at
					s.FirstSolverID = p.ContestantID
				}
			}
			stats[problemID] = s
		}
	}
	return stats
}

func markFirstSolves(entries []RankedEntry, stats map[string]problemStats) {
	for _, e := range entries {
		for problemID, rp := range e.Problems {
			if rp.AcceptedAt != nil && stats[problemID].FirstSolverID == e.ContestantID {
				rp.IsFirstSolve = true
				e.Problems[problemID] = rp
			}
		}
	}
}
