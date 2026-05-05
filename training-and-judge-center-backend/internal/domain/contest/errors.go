package contest

const (
	ErrCodeContestNotFound         = "CONTEST_NOT_FOUND"
	ErrCodeInvalidContestName      = "INVALID_CONTEST_NAME"
	ErrCodeInvalidTimeRange        = "INVALID_TIME_RANGE"
	ErrCodeStartTimeInPast         = "START_TIME_IN_PAST"
	ErrCodeInvalidPenalty          = "INVALID_PENALTY"
	ErrCodeInvalidFreezeMinutes    = "INVALID_FREEZE_MINUTES"
	ErrCodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	ErrCodeContestLocked           = "CONTEST_LOCKED"
	ErrCodeProblemNotFound         = "PROBLEM_NOT_FOUND"
	ErrCodeProblemNotPublished     = "PROBLEM_NOT_PUBLISHED"
	ErrCodeProblemAccessDenied     = "PROBLEM_ACCESS_DENIED"
)
