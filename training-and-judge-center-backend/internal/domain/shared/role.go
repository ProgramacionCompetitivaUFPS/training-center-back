package shared

import "fmt"

type Role string

const (
	RoleAdmin      Role = "ADMIN"
	RoleCoach      Role = "COACH"
	RoleContestant Role = "CONTESTANT"
)

func NewRole(value string) (Role, error) {
	switch Role(value) {
	case RoleAdmin, RoleCoach, RoleContestant:
		return Role(value), nil
	default:
		return "", fmt.Errorf("invalid role: %s", value)
	}
}

func RestoreRole(value string) Role {
	return Role(value)
}

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleCoach, RoleContestant:
		return true
	}
	return false
}

func (r Role) String() string {
	return string(r)
}
