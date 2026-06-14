package contest

import (
	"context"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

type CachedStandings struct {
	Participants []domainContest.ParticipantStanding
	LastUpdated  time.Time
}

// StandingsCache is the port for reading and writing the precomputed standings projection.
// The implementation uses Redis with stale-while-revalidate semantics.
type StandingsCache interface {
	// Get returns the cached standings, or nil if no entry exists.
	Get(ctx context.Context, contestID string) (*CachedStandings, error)
	// Set writes standings to the cache.
	Set(ctx context.Context, contestID string, data *CachedStandings) error
	// AcquireRefreshLock attempts a SET NX on a per-contest lock key.
	// Returns true if the lock was acquired (caller must recompute), false if another instance holds it.
	AcquireRefreshLock(ctx context.Context, contestID string, ttl time.Duration) (bool, error)
	// ReleaseRefreshLock deletes the lock key early (called after successful recompute).
	ReleaseRefreshLock(ctx context.Context, contestID string) error
}
