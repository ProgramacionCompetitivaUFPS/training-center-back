package testutil

import (
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func AsAdmin(id string) appshared.CurrentUser {
	return appshared.CurrentUser{ID: id, Role: shared.RoleAdmin}
}

func AsCoach(id string) appshared.CurrentUser {
	return appshared.CurrentUser{ID: id, Role: shared.RoleCoach}
}

func AsContestant(id string) appshared.CurrentUser {
	return appshared.CurrentUser{ID: id, Role: shared.RoleContestant}
}
