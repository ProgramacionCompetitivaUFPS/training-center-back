package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	memberRoleLead   = "LEAD"
	memberRoleMember = "MEMBER"
)

type MemberRole struct{ value string }

var (
	MemberRoleLead   = MemberRole{value: memberRoleLead}
	MemberRoleMember = MemberRole{value: memberRoleMember}
)

func NewMemberRole(s string) (MemberRole, error) {
	switch s {
	case memberRoleLead, memberRoleMember:
		return MemberRole{value: s}, nil
	}
	return MemberRole{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "role", Message: "invalid member role: " + s},
	})
}

func RestoreMemberRole(s string) MemberRole { return MemberRole{value: s} }
func (r MemberRole) String() string         { return r.value }
