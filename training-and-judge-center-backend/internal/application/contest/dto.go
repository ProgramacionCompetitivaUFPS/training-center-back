package contest

import (
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

type GroupDisplay struct {
	ID   string
	Name string
}

type ProblemDisplay struct {
	Slug  string
	Title string
	Order int
}

type ContestOutput struct {
	ID                string
	Name              string
	Description       *string
	StartTime         time.Time
	EndTime           time.Time
	Duration          int
	Penalty           int
	FreezeMinutes     int
	EnablePostContest bool
	Locked            bool
	Group             GroupDisplay
	Owner             UserDisplay
	Problems          []ProblemDisplay
	ProblemCount      int
	Status            string
	CreatedAt         time.Time
	UpdatedAt         *time.Time
}

func buildOutput(c *domainContest.Contest, group *GroupInfo, owner *UserDisplay, problems []ProblemDisplay, now time.Time) *ContestOutput {
	return &ContestOutput{
		ID:                c.ID(),
		Name:              c.Name().Value(),
		Description:       c.Description(),
		StartTime:         c.StartTime(),
		EndTime:           c.EndTime(),
		Duration:          c.Duration(),
		Penalty:           c.Penalty().Value(),
		FreezeMinutes:     c.FreezeMinutes(),
		EnablePostContest: c.EnablePostContest(),
		Locked:            c.Locked(),
		Group:             GroupDisplay{ID: group.ID, Name: group.Name},
		Owner:             *owner,
		Problems:          problems,
		ProblemCount:      len(problems),
		Status:            c.Status(now).String(),
		CreatedAt:         c.CreatedAt(),
		UpdatedAt:         c.UpdatedAt(),
	}
}


