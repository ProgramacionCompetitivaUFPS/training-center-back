package contest

import "context"

// ParticipantProfile is the local (contest-domain) view of a user's
// identity/location fields, used for standings filtering and display. Never
// import domain/user from here — each domain defines its own display types.
type ParticipantProfile struct {
	ID          string
	Nickname    string
	Name        string
	Country     string
	City        string
	Institution string
}

// ParticipantProfileProvider batches country/city/institution lookups by userID.
type ParticipantProfileProvider interface {
	GetProfiles(ctx context.Context, userIDs []string) (map[string]*ParticipantProfile, error)
}
