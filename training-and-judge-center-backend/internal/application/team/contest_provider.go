package team

import "context"

// ContestInfo carries the contest fields needed by team registration use cases.
type ContestInfo struct {
	ID                string
	GroupID           string
	ParticipationMode string // "INDIVIDUAL" | "TEAM" | "MIXED"
	TeamSizeMin       int
	TeamSizeMax       int
	Status            string // "SCHEDULED" | "ACTIVE" | "FINISHED"
}

func (c *ContestInfo) AllowsTeams() bool {
	return c.ParticipationMode == "TEAM" || c.ParticipationMode == "MIXED"
}

type ContestProvider interface {
	GetContestInfo(ctx context.Context, contestID string) (*ContestInfo, error)
}
