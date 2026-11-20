package contest

import (
	"sort"
	"strings"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

// RankPostcompetition orders unofficial rows by solved, last accept, name, then ID.
func RankPostcompetition(participants []domainContest.ParticipantStanding, nameOf func(contestantID, participantType string) string) []RankedEntry {
	entries := make([]RankedEntry, 0, len(participants))
	names := make(map[string]string, len(participants))

	for _, p := range participants {
		entry := RankedEntry{
			ContestantID:    p.ContestantID,
			ParticipantType: p.ParticipantType,
			Problems:        make(map[string]RankedProblem, len(p.Problems)),
		}
		for problemID, attempt := range p.Problems {
			entry.Problems[problemID] = RankedProblem{
				Attempts:   len(attempt.WrongAttemptTimes),
				AcceptedAt: attempt.AcceptedAt,
			}
			if attempt.AcceptedAt == nil {
				continue
			}
			entry.ProblemsSolved++
			if entry.LastAcceptedAt == nil || attempt.AcceptedAt.After(*entry.LastAcceptedAt) {
				t := *attempt.AcceptedAt
				entry.LastAcceptedAt = &t
			}
		}
		names[p.ContestantID] = nameOf(p.ContestantID, p.ParticipantType)
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.ProblemsSolved != b.ProblemsSolved {
			return a.ProblemsSolved > b.ProblemsSolved
		}
		if a.LastAcceptedAt != nil && b.LastAcceptedAt != nil && !a.LastAcceptedAt.Equal(*b.LastAcceptedAt) {
			return a.LastAcceptedAt.Before(*b.LastAcceptedAt)
		}
		na, nb := strings.ToLower(names[a.ContestantID]), strings.ToLower(names[b.ContestantID])
		if na != nb {
			return na < nb
		}
		return a.ContestantID < b.ContestantID
	})

	return entries
}
