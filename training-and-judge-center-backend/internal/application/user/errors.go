package user

const (
	ErrCodeInvalidCredentials     = "INVALID_CREDENTIALS"
	ErrCodeAccountDeactivated     = "ACCOUNT_DEACTIVATED"
	ErrCodeInvalidCode            = "INVALID_CODE"
	ErrCodeExpiredCode            = "EXPIRED_CODE"
	ErrCodeMaxAttemptsExceeded    = "MAX_ATTEMPTS_EXCEEDED"
	ErrCodeTooManyRequests        = "TOO_MANY_REQUESTS"
	ErrCodeEmailDeliveryFailed    = "EMAIL_DELIVERY_FAILED"
	ErrCodeUserNotFound           = "USER_NOT_FOUND"
	ErrCodeAdminProfileRestricted = "ADMIN_PROFILE_RESTRICTED"
	ErrCodeAlreadyDeactivated     = "ALREADY_DEACTIVATED"
	ErrCodeNoPendingRequest       = "NO_PENDING_REQUEST"
	ErrCodeForbidden              = "FORBIDDEN"
	ErrCodeInvalidRecoveryAttempt = "INVALID_RECOVERY_ATTEMPT"
	ErrCodeCannotSelfDeactivate   = "CANNOT_SELF_DEACTIVATE"
	ErrCodeCannotDeactivateAdmin  = "CANNOT_DEACTIVATE_ADMIN"
)
