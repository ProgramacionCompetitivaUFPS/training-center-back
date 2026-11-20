package team

import (
	"context"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ContestInfo carries the contest fields needed by team registration use cases.
type ContestInfo struct {
	ID                    string
	GroupID               string
	GroupIsVisible        bool
	ParticipationMode     string // "INDIVIDUAL" | "TEAM" | "MIXED"
	TeamSizeMin           int
	TeamSizeMax           int
	Status                string // "SCHEDULED" | "ACTIVE" | "FINISHED"
	AllowLateRegistration bool
}

func (c *ContestInfo) AllowsTeams() bool {
	return c.ParticipationMode == "TEAM" || c.ParticipationMode == "MIXED"
}

// CheckRegistrationOpen mirrors domainContest.Contest.CheckRegistrationOpen for callers that only hold a ContestInfo.
func (c *ContestInfo) CheckRegistrationOpen() error {
	switch c.Status {
	case "SCHEDULED":
		return nil
	case "ACTIVE":
		if c.AllowLateRegistration {
			return nil
		}
		return apperror.NewConflict(domainContest.ErrCodeContestAlreadyStarted, "registration is closed because the contest has already started")
	default:
		return apperror.NewConflict(domainContest.ErrCodeContestNotOpenForRegistration, "the contest has finished and is not open for registration")
	}
}

type ContestProvider interface {
	GetContestInfo(ctx context.Context, contestID string) (*ContestInfo, error)
}
