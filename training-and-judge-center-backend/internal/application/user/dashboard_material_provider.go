package user

import (
	"context"
	"time"
)

// DashboardMaterialProvider provides material data for the dashboard use case.
type DashboardMaterialProvider interface {
	// GetRecentMaterials returns up to limit published materials from all groups
	// the user belongs to, published within the last windowDays days, most
	// recently published first.
	GetRecentMaterials(ctx context.Context, userID string, limit int, windowDays int) ([]DashboardMaterial, error)
}

type DashboardMaterial struct {
	ID             string
	Title          string
	GroupID        string
	GroupName      string
	PublishedAt    time.Time
	AuthorNickname string
}
