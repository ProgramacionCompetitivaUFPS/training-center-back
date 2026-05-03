package shared

type CurrentUser struct {
	ID   string
	Role Role
}

func (c CurrentUser) IsAdmin() bool { return c.Role == RoleAdmin }
