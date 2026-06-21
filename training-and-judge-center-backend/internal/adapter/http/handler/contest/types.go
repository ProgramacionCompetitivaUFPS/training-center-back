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
	Name              string    `json:"name"`
	Description       *string   `json:"description"`
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime"`
	Penalty           *int      `json:"penalty"`
	FreezeMinutes     *int      `json:"freezeMinutes"`
	EnablePostContest bool      `json:"enablePostContest"`
	ParticipationMode string    `json:"participationMode"`
	TeamSizeMin       *int      `json:"teamSizeMin"`
	TeamSizeMax       *int      `json:"teamSizeMax"`
	Problems          []string  `json:"problems"`
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
	ParticipationMode *string               `json:"participationMode"`
	TeamSizeMin       *int                  `json:"teamSizeMin"`
	TeamSizeMax       *int                  `json:"teamSizeMax"`
}

// ── Response types ───────────────────────────────────────────────────────────

type groupDisplay struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ownerDisplay struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type problemDisplay struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

// ── GET /contests/:id response ───────────────────────────────────────────────

type problemDetail struct {
	Position    int    `json:"position"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	TimeLimit   int    `json:"timeLimit"`
	MemoryLimit int    `json:"memoryLimit"`
}

type getContestResponse struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Description       *string                 `json:"description,omitempty"`
	StartTime         string                  `json:"startTime"`
	EndTime           string                  `json:"endTime"`
	Duration          int                     `json:"duration"`
	Penalty           int                     `json:"penalty"`
	FreezeMinutes     int                     `json:"freezeMinutes"`
	EnablePostContest bool                    `json:"enablePostContest"`
	Locked            *bool                   `json:"locked,omitempty"`
	ParticipantCount  int                     `json:"participantCount"`
	IsRegistered      bool                    `json:"isRegistered"`
	Group             groupDisplay            `json:"group"`
	Owner             ownerDisplay            `json:"owner"`
	Problems          []problemDetail `json:"problems"`
	Status            string                  `json:"status"`
	CreatedAt         string                  `json:"createdAt"`
	UpdatedAt         *string                 `json:"updatedAt,omitempty"`
}

// ── GET /contests response ───────────────────────────────────────────────────

type contestListItem struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	StartTime         string  `json:"startTime"`
	EndTime           string  `json:"endTime"`
	Duration          int     `json:"duration"`
	Status            string  `json:"status"`
	Penalty           int     `json:"penalty"`
	FreezeMinutes     int     `json:"freezeMinutes"`
	EnablePostContest bool    `json:"enablePostContest"`
	ParticipantCount  int     `json:"participantCount"`
	IsRegistered      bool    `json:"isRegistered"`
	ProblemCount      int     `json:"problemCount"`
}

type pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type listContestsResponse struct {
	Items      []contestListItem `json:"items"`
	Pagination pagination        `json:"pagination"`
}

// ── Registration status response ─────────────────────────────────────────────

type registrationStatusResponse struct {
	Registered   bool    `json:"registered"`
	RegisteredAt *string `json:"registeredAt,omitempty"`
}

// ── List registrations response ───────────────────────────────────────────────

type registrationItem struct {
	Nickname     string `json:"nickname"`
	RegisteredAt string `json:"registeredAt"`
}

type registrationsPagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"totalPages"`
	HasMore    bool `json:"hasMore"`
}

type listRegistrationsResponse struct {
	Registrations []registrationItem      `json:"registrations"`
	Pagination    registrationsPagination `json:"pagination"`
}

// ── GET /contests/:id/standings response ─────────────────────────────────────

type rankedProblem struct {
	Attempts   int     `json:"attempts"`
	Penalty    int     `json:"penalty"`
	IsSolved   bool    `json:"isSolved"`
	AcceptedAt *string `json:"acceptedAt,omitempty"`
}

type rankedEntry struct {
	Rank           int                      `json:"rank"`
	ContestantID   string                   `json:"contestantId"`
	ProblemsSolved int                      `json:"problemsSolved"`
	TotalPenalty   int                      `json:"totalPenalty"`
	LastAcceptedAt *string                  `json:"lastAcceptedAt,omitempty"`
	Problems       map[string]rankedProblem `json:"problems"`
}

type standingsMeta struct {
	LastUpdated   string  `json:"lastUpdated"`
	IsFrozen      bool    `json:"isFrozen"`
	FrozenAt      *string `json:"frozenAt,omitempty"`
	ContestStatus string  `json:"contestStatus"`
}

type standingsPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type getStandingsResponse struct {
	Entries    []rankedEntry       `json:"entries"`
	Pagination standingsPagination `json:"pagination"`
	Meta       standingsMeta       `json:"meta"`
}

// ── GET /contests/:id/submissions response ───────────────────────────────────

type submissionProblem struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

type submissionSubmitter struct {
	Type     string   `json:"type"`
	Nickname string   `json:"nickname,omitempty"`
	Name     string   `json:"name,omitempty"`
	TeamID   string   `json:"teamId,omitempty"`
	TeamName string   `json:"teamName,omitempty"`
	Members  []string `json:"members,omitempty"`
}

type submissionItem struct {
	ID          string              `json:"id"`
	Problem     submissionProblem   `json:"problem"`
	SubmittedBy submissionSubmitter `json:"submittedBy"`
	Language    string              `json:"language"`
	Status      string              `json:"status"`
	SubmittedAt string              `json:"submittedAt"`
	JudgedAt    *string             `json:"judgedAt,omitempty"`
	TimeMs      *int                `json:"timeMs,omitempty"`
	MemoryKb    *int                `json:"memoryKb,omitempty"`
	Phase       *string             `json:"phase,omitempty"`
}

type submissionsContestMeta struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	InFreeze   bool    `json:"inFreeze"`
	FreezeTime *string `json:"freezeTime,omitempty"`
}

type listSubmissionsResponse struct {
	Contest     submissionsContestMeta `json:"contest"`
	Submissions []submissionItem       `json:"submissions"`
	Pagination  standingsPagination    `json:"pagination"`
}

// ── Create/Update response ───────────────────────────────────────────────────

type contestResponse struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       *string          `json:"description,omitempty"`
	StartTime         string           `json:"startTime"`
	EndTime           string           `json:"endTime"`
	Duration          int              `json:"duration"`
	Penalty           int              `json:"penalty"`
	FreezeMinutes     int              `json:"freezeMinutes"`
	EnablePostContest bool             `json:"enablePostContest"`
	Locked            bool             `json:"locked"`
	ParticipationMode string           `json:"participationMode"`
	TeamSizeMin       int              `json:"teamSizeMin"`
	TeamSizeMax       int              `json:"teamSizeMax"`
	Group             groupDisplay     `json:"group"`
	Owner             ownerDisplay     `json:"owner"`
	Problems          []problemDisplay `json:"problems"`
	ProblemCount      int              `json:"problemCount"`
	Status            string           `json:"status"`
	CreatedAt         string           `json:"createdAt"`
	UpdatedAt         *string          `json:"updatedAt,omitempty"`
}
