package contest

import "time"

// ── Request types ────────────────────────────────────────────────────────────

type createContestRequest struct {
	Name              string   `json:"name"`
	Description       *string  `json:"description"`
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime"`
	Penalty           *int     `json:"penalty"`
	FreezeMinutes     *int     `json:"freezeMinutes"`
	EnablePostContest bool     `json:"enablePostContest"`
	Problems          []string `json:"problems"`
}

type problemOrderRequest struct {
	Slug  string `json:"slug"`
	Order int    `json:"order"`
}

type updateContestRequest struct {
	Name              *string               `json:"name"`
	Description       *string               `json:"description"`
	StartTime         *time.Time            `json:"startTime"`
	EndTime           *time.Time            `json:"endTime"`
	Penalty           *int                  `json:"penalty"`
	FreezeMinutes     *int                  `json:"freezeMinutes"`
	EnablePostContest *bool                 `json:"enablePostContest"`
	Problems          *[]problemOrderRequest `json:"problems"`
	Locked            *bool                 `json:"locked"`
}

// ── Response types ───────────────────────────────────────────────────────────

type groupDisplay struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ownerDisplay struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type problemDisplay struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

type contestResponse struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       *string        `json:"description,omitempty"`
	StartTime         string         `json:"startTime"`
	EndTime           string         `json:"endTime"`
	Duration          int            `json:"duration"`
	Penalty           int            `json:"penalty"`
	FreezeMinutes     int            `json:"freezeMinutes"`
	EnablePostContest bool           `json:"enablePostContest"`
	Locked            bool           `json:"locked"`
	Group             groupDisplay   `json:"group"`
	Owner             ownerDisplay   `json:"owner"`
	Problems          []problemDisplay `json:"problems"`
	ProblemCount      int            `json:"problemCount"`
	Status            string         `json:"status"`
	CreatedAt         string         `json:"createdAt"`
	UpdatedAt         *string        `json:"updatedAt,omitempty"`
}
