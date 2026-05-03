package group

import "github.com/training-judge-center/backend/pkg/apperror"

type MemberRole struct{ value string }

var (
	MemberRoleLead   = MemberRole{value: "LEAD"}
	MemberRoleMember = MemberRole{value: "MEMBER"}
)

func NewMemberRole(s string) (MemberRole, error) {
	switch s {
	case "LEAD", "MEMBER":
		return MemberRole{value: s}, nil
	}
	return MemberRole{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "role", Message: "invalid member role: " + s},
	})
}

func RestoreMemberRole(s string) MemberRole { return MemberRole{value: s} }
func (r MemberRole) String() string         { return r.value }
