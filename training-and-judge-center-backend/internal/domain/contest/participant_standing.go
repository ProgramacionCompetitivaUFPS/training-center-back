package contest

import "time"

// ProblemAttempt holds raw submission data for one problem of one participant.
// Penalty and freeze visibility are computed at ranking time from timestamps,
// so both stay correct if contestPenalty or freezeMinutes change without cache invalidation.
type ProblemAttempt struct {
	// Timestamps of wrong submissions (WA/TLE/MLE/RE) before the first AC.
	// CE and SE excluded per ICPC rules.
	WrongAttemptTimes []time.Time
	AcceptedAt        *time.Time // nil = not yet solved
}

// ParticipantStanding is the per-participant projection cached in Redis.
// It is the input to RankStandings.
type ParticipantStanding struct {
	ContestantID    string
	ParticipantType string // "INDIVIDUAL" or "TEAM"
	Problems        map[string]ProblemAttempt // key: problemID
}
