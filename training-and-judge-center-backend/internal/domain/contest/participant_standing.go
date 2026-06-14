package contest

import "time"

// ProblemAttempt holds raw submission data for one problem of one participant.
// Penalty is computed at ranking time (not stored) so it stays correct if contestPenalty changes.
type ProblemAttempt struct {
	// Wrong submissions before the first AC (CE and SE excluded per ICPC rules).
	Attempts   int
	AcceptedAt *time.Time // nil = not yet solved
}

// ParticipantStanding is the per-participant projection cached in Redis.
// It is the input to RankStandings.
type ParticipantStanding struct {
	ContestantID string
	Problems     map[string]ProblemAttempt // key: problemID
}
