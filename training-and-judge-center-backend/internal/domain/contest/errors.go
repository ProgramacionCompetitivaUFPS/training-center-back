package contest

const (
	ErrCodeContestNotFound               = "CONTEST_NOT_FOUND"
	ErrCodeContestConflict               = "CONTEST_CONFLICT"
	ErrCodeContestLocked                 = "CONTEST_LOCKED"
	ErrCodeStartTimeInPast               = "START_TIME_IN_PAST"
	ErrCodeEndTimeInPast                 = "END_TIME_IN_PAST"
	ErrCodeInvalidTimeRange              = "INVALID_TIME_RANGE"
	ErrCodeContestAlreadyStarted         = "CONTEST_ALREADY_STARTED"
	ErrCodeContestNotOpenForRegistration = "CONTEST_NOT_OPEN_FOR_REGISTRATION"
	ErrCodeCannotUnregisterAfterStart    = "CANNOT_UNREGISTER_AFTER_START"
	ErrCodeNotRegistered                 = "NOT_REGISTERED"
	ErrCodeTeamAlreadyRegistered         = "TEAM_ALREADY_REGISTERED"
	ErrCodeTeamRegistrationNotFound      = "TEAM_REGISTRATION_NOT_FOUND"
	ErrCodeContestHasTeamRegistrations   = "CONTEST_HAS_TEAM_REGISTRATIONS"
	ErrCodeContestIsActive               = "CONTEST_IS_ACTIVE"
	ErrCodeContestProblemsLocked         = "CONTEST_PROBLEMS_LOCKED"
	ErrCodeFreezeTooLong                 = "FREEZE_TOO_LONG"
	ErrCodeRegistrationBusy              = "REGISTRATION_BUSY"
	ErrCodeInvalidProblemColor           = "INVALID_PROBLEM_COLOR"
	ErrCodeContestHasNoFreeze            = "CONTEST_HAS_NO_FREEZE"
	ErrCodeContestNotFinished            = "CONTEST_NOT_FINISHED"
)
