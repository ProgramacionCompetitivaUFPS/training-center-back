package user

const (
	ErrCodeEmailConflict              = "EMAIL_CONFLICT"
	ErrCodeNicknameConflict           = "NICKNAME_CONFLICT"
	ErrCodeEmailChangeNotPending      = "EMAIL_CHANGE_NOT_PENDING"
	ErrCodePasswordRecoveryNotPending = "PASSWORD_RECOVERY_NOT_PENDING"
	ErrCodeAlreadyDeactivated         = "ALREADY_DEACTIVATED"
	ErrCodeCannotUpdateDeactivated    = "CANNOT_UPDATE_DEACTIVATED"
	ErrCodeCannotAssignAdminRole      = "CANNOT_ASSIGN_ADMIN_ROLE"
	ErrCodeUserNotFound               = "USER_NOT_FOUND"
	ErrCodeRefreshTokenAlreadyRevoked = "REFRESH_TOKEN_ALREADY_REVOKED"
	ErrCodeRefreshTokenExpired        = "REFRESH_TOKEN_EXPIRED"
	ErrCodeOAuthIdentityConflict      = "OAUTH_IDENTITY_CONFLICT"
	ErrCodeOAuthIdentityAlreadyLinked = "OAUTH_IDENTITY_ALREADY_LINKED"
	ErrCodeOAuthIdentityNotFound      = "OAUTH_IDENTITY_NOT_FOUND"
)
