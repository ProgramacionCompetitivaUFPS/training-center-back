package user

import "context"

// TopicStatsProvider computes the breakdown of the user's solved problems by tag.
type TopicStatsProvider interface {
	// GetTopicBreakdown returns, for each tag present on at least one problem
	// the user solved, the count of unique problems solved carrying that tag,
	// ordered by count descending.
	GetTopicBreakdown(ctx context.Context, userID string) ([]TopicStat, error)
}

type TopicStat struct {
	Tag    string
	Solved int
}
