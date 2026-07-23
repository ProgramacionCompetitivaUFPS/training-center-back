package group

import (
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)

type GroupDTO struct {
	ID          string
	Name        string
	Description *string
	JoinPolicy  string
	Visibility  string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MemberDTO struct {
	Role     string
	JoinedAt time.Time
}

type JoinRequestDTO struct {
	ID              string
	GroupID         string
	RequesterUserID string
	Status          string
	Message         *string
	CreatedAt       time.Time
}

type GroupInvitationDTO struct {
	ID        string
	GroupID   string
	InviteeID *string // nil for a general (link-only) invitation
	InvitedBy string
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func groupToDTO(g *domainGroup.Group) GroupDTO {
	return GroupDTO{
		ID:          g.ID(),
		Name:        g.Name().String(),
		Description: g.Description(),
		JoinPolicy:  g.JoinPolicy().String(),
		Visibility:  g.Visibility().String(),
		IsDefault:   g.IsDefault(),
		CreatedAt:   g.CreatedAt(),
		UpdatedAt:   g.UpdatedAt(),
	}
}

func memberToDTO(m *domainGroup.GroupMember) MemberDTO {
	return MemberDTO{Role: m.Role().String(), JoinedAt: m.JoinedAt()}
}

func joinRequestToDTO(r *domainGroup.JoinRequest) JoinRequestDTO {
	return JoinRequestDTO{
		ID:              r.ID(),
		GroupID:         r.GroupID(),
		RequesterUserID: r.RequesterUserID().Value(),
		Status:          r.Status().String(),
		Message:         r.Message(),
		CreatedAt:       r.CreatedAt(),
	}
}

func groupInvitationToDTO(inv *domainGroup.GroupInvitation) GroupInvitationDTO {
	var inviteeID *string
	if id := inv.InviteeID(); id != nil {
		v := id.Value()
		inviteeID = &v
	}
	return GroupInvitationDTO{
		ID:        inv.ID(),
		GroupID:   inv.GroupID(),
		InviteeID: inviteeID,
		InvitedBy: inv.InvitedBy().Value(),
		Status:    inv.Status().String(),
		ExpiresAt: inv.ExpiresAt(),
		CreatedAt: inv.CreatedAt(),
	}
}
