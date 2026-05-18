package contest

import "time"

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
