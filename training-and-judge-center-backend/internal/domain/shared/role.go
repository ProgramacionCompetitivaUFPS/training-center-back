package shared

import "github.com/training-judge-center/backend/pkg/apperror"

type Role struct{ value string }

var (
	RoleAdmin      = Role{value: "ADMIN"}
	RoleCoach      = Role{value: "COACH"}
	RoleContestant = Role{value: "CONTESTANT"}
)

func NewRole(value string) (Role, error) {
	switch value {
	case "ADMIN", "COACH", "CONTESTANT":
		return Role{value: value}, nil
	default:
		return Role{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "role", Message: "invalid role: " + value},
		})
	}
}

func RestoreRole(value string) Role {
	return Role{value: value}
}

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleCoach, RoleContestant:
		return true
	}
	return false
}

func (r Role) String() string {
	return r.value
}
