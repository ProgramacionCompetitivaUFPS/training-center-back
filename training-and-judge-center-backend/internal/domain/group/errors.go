package group

const (
	ErrCodeGroupNotFound           = "GROUP_NOT_FOUND"
	ErrCodeRequestNotFound         = "REQUEST_NOT_FOUND"
	ErrCodeRequestAlreadyProcessed = "REQUEST_ALREADY_PROCESSED"
	ErrCodeNameAlreadyExists       = "NAME_ALREADY_EXISTS"
	ErrCodeRoleUnchanged           = "ROLE_UNCHANGED"
	ErrCodeAlreadyMember           = "ALREADY_MEMBER"
	ErrCodeRequestAlreadyPending   = "REQUEST_ALREADY_PENDING"
	ErrCodeInvalidInviteToken      = "INVALID_INVITE_TOKEN"
	ErrCodeExpiredInviteToken      = "EXPIRED_INVITE_TOKEN"

	ErrCodeNotAMember                      = "NOT_A_MEMBER"
	ErrCodeInvalidLeadAssignment           = "INVALID_LEAD_ASSIGNMENT"
	ErrCodeCannotRemoveLastLead            = "CANNOT_REMOVE_LAST_LEAD"
	ErrCodeCannotRemoveFromGlobalGroup     = "CANNOT_REMOVE_FROM_GLOBAL_GROUP"
	ErrCodeCannotRemoveAdminFromGlobalGroup = "CANNOT_REMOVE_ADMIN_FROM_GLOBAL_GROUP"
	ErrCodeCannotLeaveGlobalGroup          = "CANNOT_LEAVE_GLOBAL_GROUP"
	ErrCodeCannotLeaveAsLastLead           = "CANNOT_LEAVE_AS_LAST_LEAD"
)
