package contest

import "context"

// TeamDisplay is the local (contest-domain) view of a team's name, used to
// enrich TEAM standings entries. Never import domain/team from here — each
// domain defines its own display types.
type TeamDisplay struct {
	ID   string
	Name string
}

// TeamDisplayProvider batches team name lookups by teamID.
type TeamDisplayProvider interface {
	GetDisplays(ctx context.Context, teamIDs []string) (map[string]*TeamDisplay, error)
}
