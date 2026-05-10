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
)

