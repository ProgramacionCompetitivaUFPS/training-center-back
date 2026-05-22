package shared

import domainshared "github.com/training-judge-center/backend/internal/domain/shared"

type CurrentUser struct {
	ID   string
	Role domainshared.Role
}

func (c CurrentUser) IsAdmin() bool { return c.Role == domainshared.RoleAdmin }
