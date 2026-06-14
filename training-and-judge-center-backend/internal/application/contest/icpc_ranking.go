package contest

import (
	"sort"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

// RankStandings computes ICPC standings from raw participant data.
//
// Sort order: problemsSolved DESC → totalPenalty ASC → lastAcceptedAt ASC.
// Penalty per problem: acceptedMinutes + wrongAttempts × contestPenalty.
// If freezeTime is non-nil, submissions after that point are invisible:
// wrong attempts are not counted and accepted submissions are hidden.
func RankStandings(
	participants []domainContest.ParticipantStanding,
	contestStart time.Time,
	contestPenalty int,
	freezeTime *time.Time,
) []RankedEntry {
	entries := make([]RankedEntry, 0, len(participants))

	for _, p := range participants {
		entry := RankedEntry{
			ContestantID: p.ContestantID,
			Problems:     make(map[string]RankedProblem, len(p.Problems)),
		}

		var lastAccepted *time.Time

		for problemID, attempt := range p.Problems {
			// Count only pre-freeze wrong attempts; post-freeze ones are invisible.
			preFreeze := 0
			for _, t := range attempt.WrongAttemptTimes {
				if freezeTime == nil || !t.After(*freezeTime) {
					preFreeze++
				}
			}

			rp := RankedProblem{Attempts: preFreeze}

			if attempt.AcceptedAt != nil && (freezeTime == nil || !attempt.AcceptedAt.After(*freezeTime)) {
				minutes := int(attempt.AcceptedAt.Sub(contestStart).Minutes())
				rp.Penalty = minutes + preFreeze*contestPenalty
				rp.AcceptedAt = attempt.AcceptedAt
				entry.ProblemsSolved++
				entry.TotalPenalty += rp.Penalty
				if lastAccepted == nil || attempt.AcceptedAt.After(*lastAccepted) {
					t := *attempt.AcceptedAt
					lastAccepted = &t
				}
			}

			entry.Problems[problemID] = rp
		}

		entry.LastAcceptedAt = lastAccepted
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.ProblemsSolved != b.ProblemsSolved {
			return a.ProblemsSolved > b.ProblemsSolved
		}
		if a.TotalPenalty != b.TotalPenalty {
			return a.TotalPenalty < b.TotalPenalty
		}
		if a.LastAcceptedAt == nil {
			return false
		}
		if b.LastAcceptedAt == nil {
			return true
		}
		return a.LastAcceptedAt.Before(*b.LastAcceptedAt)
	})

	// Assign ranks; tied entries share the same rank number.
	for i := range entries {
		if i == 0 {
			entries[i].Rank = 1
			continue
		}
		prev := entries[i-1]
		curr := entries[i]
		sameTie := prev.ProblemsSolved == curr.ProblemsSolved &&
			prev.TotalPenalty == curr.TotalPenalty &&
			lastAcceptedEqual(prev.LastAcceptedAt, curr.LastAcceptedAt)
		if sameTie {
			entries[i].Rank = prev.Rank
		} else {
			entries[i].Rank = i + 1
		}
	}

	return entries
}

func lastAcceptedEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
