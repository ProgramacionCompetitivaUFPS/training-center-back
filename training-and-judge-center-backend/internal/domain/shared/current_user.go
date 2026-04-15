package shared

const (
	RoleAdmin      = "ADMIN"
	RoleCoach      = "COACH"
	RoleContestant = "CONTESTANT"
)

type CurrentUser struct {
	ID   string
	Role string
}

func (c CurrentUser) IsAdmin() bool { return c.Role == RoleAdmin }
