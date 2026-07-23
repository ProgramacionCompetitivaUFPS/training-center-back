package user

import "context"

// SubmissionStatsProvider computes the user's all-time submission counts.
type SubmissionStatsProvider interface {
	// GetSubmissionCounts returns the total number of submissions ever made
	// by the user, and how many of those had an ACCEPTED verdict.
	GetSubmissionCounts(ctx context.Context, userID string) (total int, accepted int, err error)
}
