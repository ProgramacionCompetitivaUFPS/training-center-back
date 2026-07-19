package user

import "context"

type TopicStatsProvider interface {
	// GetTopicBreakdown counts solved problems (ACCEPTED) per tag, ordered by count descending.
	GetTopicBreakdown(ctx context.Context, userID string) ([]TopicStat, error)
}

type TopicStat struct {
	Tag    string
	Solved int
}
