package contest

import (
	"encoding/json"
	"time"
)

// nullableString distinguishes three JSON states for a string field:
//   - key omitted  → Present=false
//   - key: null    → Present=true, Value=nil
//   - key: "text"  → Present=true, Value=&"text"
type nullableString struct {
	Present bool
	Value   *string
}

func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.Present = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}

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
	Description       nullableString        `json:"description"`
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
