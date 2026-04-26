package group

import "github.com/training-judge-center/backend/pkg/apperror"

type MemberRole string

const (
	MemberRoleLead   MemberRole = "LEAD"
	MemberRoleMember MemberRole = "MEMBER"
)

func NewMemberRole(s string) (MemberRole, error) {
	switch MemberRole(s) {
	case MemberRoleLead, MemberRoleMember:
		return MemberRole(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "role", Message: "invalid member role: " + s},
	})
}

func RestoreMemberRole(s string) MemberRole { return MemberRole(s) }
func (r MemberRole) String() string         { return string(r) }
