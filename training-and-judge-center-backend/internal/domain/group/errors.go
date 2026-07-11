package group

const (
	ErrCodeGroupNotFound           = "GROUP_NOT_FOUND"
	ErrCodeRequestNotFound         = "REQUEST_NOT_FOUND"
	ErrCodeRequestAlreadyProcessed = "REQUEST_ALREADY_PROCESSED"
	ErrCodeNameAlreadyExists       = "NAME_ALREADY_EXISTS"
	ErrCodeRoleUnchanged           = "ROLE_UNCHANGED"
	ErrCodeAlreadyMember           = "ALREADY_MEMBER"
	ErrCodeRequestAlreadyPending   = "REQUEST_ALREADY_PENDING"

	ErrCodeInvitationNotFound         = "INVITATION_NOT_FOUND"
	ErrCodeInvitationAlreadyProcessed = "INVITATION_ALREADY_PROCESSED"
	ErrCodeInvitationExpired          = "INVITATION_EXPIRED"

	ErrCodeNotAMember                       = "NOT_A_MEMBER"
	ErrCodeInvalidLeadAssignment            = "INVALID_LEAD_ASSIGNMENT"
	ErrCodeCannotRemoveLastLead             = "CANNOT_REMOVE_LAST_LEAD"
	ErrCodeCannotAddToGlobalGroup           = "CANNOT_ADD_TO_GLOBAL_GROUP"
	ErrCodeCannotRemoveFromGlobalGroup      = "CANNOT_REMOVE_FROM_GLOBAL_GROUP"
	ErrCodeCannotRemoveAdminFromGlobalGroup = "CANNOT_REMOVE_ADMIN_FROM_GLOBAL_GROUP"
	ErrCodeCannotLeaveGlobalGroup           = "CANNOT_LEAVE_GLOBAL_GROUP"
	ErrCodeCannotLeaveAsLastLead            = "CANNOT_LEAVE_AS_LAST_LEAD"

	ErrCodeCannotDeleteGlobalGroup  = "CANNOT_DELETE_GLOBAL_GROUP"
	ErrCodeCannotModifyGlobalGroup  = "CANNOT_MODIFY_GLOBAL_GROUP"
	ErrCodeInvalidPolicyCombination = "INVALID_POLICY_COMBINATION"
)
