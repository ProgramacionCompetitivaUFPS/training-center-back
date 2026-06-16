package team

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Team struct {
	id        string
	name      TeamName
	createdBy shared.UserID
	createdAt time.Time
}

func NewTeam(id string, name TeamName, createdBy shared.UserID, now time.Time) (*Team, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	return &Team{
		id:        id,
		name:      name,
		createdBy: createdBy,
		createdAt: now.UTC(),
	}, nil
}

func RestoreTeam(id string, name TeamName, createdBy shared.UserID, createdAt time.Time) *Team {
	return &Team{
		id:        id,
		name:      name,
		createdBy: createdBy,
		createdAt: createdAt,
	}
}

func (t *Team) ID() string               { return t.id }
func (t *Team) Name() TeamName           { return t.name }
func (t *Team) CreatedBy() shared.UserID { return t.createdBy }
func (t *Team) CreatedAt() time.Time     { return t.createdAt }
