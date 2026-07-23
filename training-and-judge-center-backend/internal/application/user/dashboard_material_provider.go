package user

import "context"

type DashboardMaterialProvider interface {
	// GetRecentMaterialsCount returns the count of published materials from
	// all groups the user belongs to, published within the last windowDays days.
	GetRecentMaterialsCount(ctx context.Context, userID string, windowDays int) (int, error)
}
