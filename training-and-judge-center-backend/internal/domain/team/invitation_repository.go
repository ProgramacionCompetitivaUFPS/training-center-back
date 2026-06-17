package team

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type InvitationRepository interface {
	Save(ctx context.Context, inv *TeamInvitation) error
	FindByID(ctx context.Context, id string) (*TeamInvitation, error)
	FindByTeamAndInvitee(ctx context.Context, teamID string, inviteeID shared.UserID) (*TeamInvitation, error)
	FindByInvitee(ctx context.Context, inviteeID shared.UserID) ([]*TeamInvitation, error)
	Delete(ctx context.Context, id string) error
}
