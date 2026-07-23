package submission

const (
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	ErrCodeNotRegistered       = "NOT_REGISTERED"
	ErrCodeProblemNotInContest = "PROBLEM_NOT_IN_CONTEST"
	ErrCodeContestNotStarted   = "CONTEST_NOT_STARTED"
	ErrCodeContestFinished     = "CONTEST_FINISHED"

	ErrCodeNoRejudgeNeeded              = "NO_REJUDGE_NEEDED"
	ErrCodeCannotRejudgeInActiveContest = "CANNOT_REJUDGE_IN_ACTIVE_CONTEST"
)
