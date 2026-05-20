package contest

const (
	ErrCodeContestNotFound    = "CONTEST_NOT_FOUND"
	ErrCodeContestConflict    = "CONTEST_CONFLICT"
	ErrCodeContestLocked      = "CONTEST_LOCKED"
	ErrCodeStartTimeInPast    = "START_TIME_IN_PAST"
	ErrCodeEndTimeInPast      = "END_TIME_IN_PAST"
	ErrCodeInvalidTimeRange   = "INVALID_TIME_RANGE"
	ErrCodeAlreadyRegistered  = "ALREADY_REGISTERED"
	ErrCodeRegistrationClosed = "REGISTRATION_CLOSED"
)
