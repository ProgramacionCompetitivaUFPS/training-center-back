package user

import (
	"context"
	"time"
)

// DashboardContestProvider provides contest data for the dashboard use case.
type DashboardContestProvider interface {
	// GetUpcomingContests returns up to limit contests the user is registered to
	// that have not started yet, ordered by start_time ASC.
	GetUpcomingContests(ctx context.Context, userID string, limit int) ([]DashboardContest, error)
	// GetActiveContests returns up to limit contests the user is registered to
	// that are currently running, ordered by start_time ASC.
	GetActiveContests(ctx context.Context, userID string, limit int) ([]DashboardContest, error)
	// GetFinishedContestResults returns up to limit finished contests the user
	// participated in, most recently ended first, with score and position.
	GetFinishedContestResults(ctx context.Context, userID string, limit int) ([]DashboardContestResult, error)
}

type DashboardContest struct {
	ID              string
	Name            string
	StartTime       time.Time
	DurationMinutes int
	GroupID         string
	GroupName       string
}

type DashboardContestResult struct {
	ContestID      string
	ContestName    string
	Position       int
	ProblemsSolved int
	Penalty        int
}
