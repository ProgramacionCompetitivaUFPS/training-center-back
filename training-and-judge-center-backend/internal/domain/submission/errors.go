package submission

const (
	ErrCodeInvalidTransition  = "INVALID_STATUS_TRANSITION"
	ErrCodeInvalidLanguage    = "INVALID_LANGUAGE"
	ErrCodeSubmissionNotFound = "SUBMISSION_NOT_FOUND"
	ErrCodeAccessDenied       = "ACCESS_DENIED"

	ErrCodeProblemNotPublished  = "PROBLEM_NOT_PUBLISHED"
	ErrCodeProblemNotAccessible = "PROBLEM_NOT_ACCESSIBLE"
	ErrCodeDuplicateSubmission  = "DUPLICATE_SUBMISSION"
	ErrCodeRateLimitExceeded    = "RATE_LIMIT_EXCEEDED"
	ErrCodeFileTooLarge         = "FILE_TOO_LARGE"
	ErrCodeCompilerMismatch     = "COMPILER_MISMATCH"
	ErrCodeNoTestCases          = "NO_TEST_CASES"

	ErrCodeNotRegistered      = "NOT_REGISTERED"
	ErrCodeProblemNotInContest = "PROBLEM_NOT_IN_CONTEST"
	ErrCodeContestNotStarted  = "CONTEST_NOT_STARTED"
	ErrCodeContestFinished    = "CONTEST_FINISHED"
)
