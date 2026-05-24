package contest

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestRegistration struct {
	id           string
	contestID    string
	userID       string
	registeredAt time.Time
}

func NewContestRegistration(id, contestID, userID string, now time.Time) (*ContestRegistration, error) {
	if id == "" || contestID == "" || userID == "" || now.IsZero() {
		return nil, apperror.NewInternal()
	}
	return &ContestRegistration{
		id:           id,
		contestID:    contestID,
		userID:       userID,
		registeredAt: now.UTC(),
	}, nil
}

func RestoreContestRegistration(id, contestID, userID string, registeredAt time.Time) *ContestRegistration {
	return &ContestRegistration{
		id:           id,
		contestID:    contestID,
		userID:       userID,
		registeredAt: registeredAt.UTC(),
	}
}

func (r *ContestRegistration) ID() string              { return r.id }
func (r *ContestRegistration) ContestID() string       { return r.contestID }
func (r *ContestRegistration) UserID() string          { return r.userID }
func (r *ContestRegistration) RegisteredAt() time.Time { return r.registeredAt }
