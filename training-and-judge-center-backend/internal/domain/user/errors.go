package user

const (
	ErrCodeEmailConflict              = "EMAIL_CONFLICT"
	ErrCodeNicknameConflict           = "NICKNAME_CONFLICT"
	ErrCodeCannotSelfDeactivate       = "CANNOT_SELF_DEACTIVATE"
	ErrCodeCannotDeactivateAdmin      = "CANNOT_DEACTIVATE_ADMIN"
	ErrCodeEmailChangeNotPending      = "EMAIL_CHANGE_NOT_PENDING"
	ErrCodePasswordRecoveryNotPending = "PASSWORD_RECOVERY_NOT_PENDING"
	ErrCodeAlreadyDeactivated         = "ALREADY_DEACTIVATED"
	ErrCodeCannotUpdateDeactivated    = "CANNOT_UPDATE_DEACTIVATED"
	ErrCodeCannotAssignAdminRole      = "CANNOT_ASSIGN_ADMIN_ROLE"
)

