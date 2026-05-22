package group

type InvitationClaims struct {
	GroupID string
}

type InvitationTokenService interface {
	GenerateInviteToken(groupID, inviterID string) (string, error)
	ValidateInviteToken(token string) (*InvitationClaims, error)
}
