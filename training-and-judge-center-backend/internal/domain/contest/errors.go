package contest

const (
	ErrCodeContestNotFound      = "CONTEST_NOT_FOUND"
	ErrCodeContestConflict      = "CONTEST_CONFLICT"
	ErrCodeContestLocked        = "CONTEST_LOCKED"
	ErrCodeInvalidContestName   = "INVALID_CONTEST_NAME"
	ErrCodeStartTimeInPast      = "START_TIME_IN_PAST"
	ErrCodeInvalidTimeRange     = "INVALID_TIME_RANGE"
	ErrCodeInvalidPenalty       = "INVALID_PENALTY"
	ErrCodeInvalidFreezeMinutes = "INVALID_FREEZE_MINUTES"
	ErrCodeDescriptionTooLong   = "DESCRIPTION_TOO_LONG"
)
