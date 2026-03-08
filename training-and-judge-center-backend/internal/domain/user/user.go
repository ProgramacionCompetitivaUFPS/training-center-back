package user

type Role string

const (
	RoleAdmin      Role = "ADMIN"
	RoleCoach      Role = "COACH"
	RoleContestant Role = "CONTESTANT"
)

type CurrentUser struct {
	ID   string
	Role Role
}

type Display struct {
	Nickname string
	Name     string
}
