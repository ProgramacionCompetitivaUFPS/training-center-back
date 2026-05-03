package group

import "context"

type UserDisplay struct {
	Nickname string
	Name     string
	Email    string
}

type UserProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}

type PreferencesReader interface {
	HideGlobalGroup(ctx context.Context, userID string) (bool, error)
}

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type InvitationClaims struct {
	GroupID string
}

type InvitationTokenService interface {
	GenerateInviteToken(groupID, inviterID string) (string, error)
	ValidateInviteToken(token string) (*InvitationClaims, error)
}
