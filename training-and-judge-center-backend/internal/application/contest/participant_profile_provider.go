package contest

import "context"

// ParticipantProfile is the local (contest-domain) view of a user's
// location/affiliation fields, used only for standings filtering. Never
// import domain/user from here — each domain defines its own display types.
type ParticipantProfile struct {
	ID          string
	Country     string
	City        string
	Institution string
}

// ParticipantProfileProvider batches country/city/institution lookups by userID.
type ParticipantProfileProvider interface {
	GetProfiles(ctx context.Context, userIDs []string) (map[string]*ParticipantProfile, error)
}
