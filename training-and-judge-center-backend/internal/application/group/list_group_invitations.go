package group

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListGroupInvitationsInput struct {
	GroupID     string
	Status      string
	Page        int
	Limit       int
	CurrentUser appshared.CurrentUser
}

type InvitationDetail struct {
	Invitation GroupInvitationDTO
	// EffectiveStatus mirrors Invitation.Status, except a still-PENDING row
	// whose TTL has already elapsed is reported as EXPIRED here without
	// persisting the transition (lazy expiration is only written on accept).
	EffectiveStatus string
	Invitee         *UserDisplay // nil for a general (link-only) invitation
	InvitedBy       *UserDisplay
}

type ListGroupInvitationsOutput struct {
	Invitations []InvitationDetail
	Total       int
	TotalPages  int
}

type ListGroupInvitationsUseCase struct {
	memberRepo     domainGroup.MemberRepository
	invitationRepo domainGroup.InvitationRepository
	userProvider   UserProvider
}

func NewListGroupInvitationsUseCase(
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	userProvider UserProvider,
) *ListGroupInvitationsUseCase {
	return &ListGroupInvitationsUseCase{memberRepo: memberRepo, invitationRepo: invitationRepo, userProvider: userProvider}
}

func (uc *ListGroupInvitationsUseCase) Execute(ctx context.Context, input ListGroupInvitationsInput) (*ListGroupInvitationsOutput, error) {
	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if err := appshared.ValidatePagination(input.Page, input.Limit, maxRequestsPageLimit); err != nil {
		return nil, err
	}
	page := input.Page
	limit := input.Limit

	var statusFilter *domainGroup.InvitationStatus
	if input.Status != "" {
		s, err := domainGroup.NewInvitationStatus(input.Status)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "status", Message: "invalid status filter; must be PENDING, ACCEPTED, REVOKED, or EXPIRED"},
			})
		}
		statusFilter = &s
	} else {
		pending := domainGroup.InvitationStatusPending
		statusFilter = &pending
	}

	invitations, total, err := uc.invitationRepo.FindByGroup(ctx, input.GroupID, domainGroup.InvitationFilters{
		Status: statusFilter,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	seenUserIDs := make(map[string]struct{}, len(invitations)*2)
	userIDs := make([]string, 0, len(invitations)*2)
	addUserID := func(id string) {
		if _, ok := seenUserIDs[id]; ok {
			return
		}
		seenUserIDs[id] = struct{}{}
		userIDs = append(userIDs, id)
	}
	for _, inv := range invitations {
		if inv.HasInvitee() {
			addUserID(inv.InviteeID().Value())
		}
		addUserID(inv.InvitedBy().Value())
	}
	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	details := make([]InvitationDetail, 0, len(invitations))
	for _, inv := range invitations {
		var invitee *UserDisplay
		if inv.HasInvitee() {
			invitee = displays[inv.InviteeID().Value()]
			if invitee == nil {
				slog.ErrorContext(ctx, "user display not found for invitation invitee", "user_id", inv.InviteeID().Value())
				invitee = &UserDisplay{Nickname: "unknown"}
			}
		}
		invitedBy := displays[inv.InvitedBy().Value()]
		if invitedBy == nil {
			slog.ErrorContext(ctx, "user display not found for invitation inviter", "user_id", inv.InvitedBy().Value())
			invitedBy = &UserDisplay{Nickname: "unknown"}
		}

		effectiveStatus := inv.Status().String()
		if inv.IsPending() && inv.IsExpired(now) {
			effectiveStatus = domainGroup.InvitationStatusExpired.String()
		}

		details = append(details, InvitationDetail{
			Invitation:      groupInvitationToDTO(inv),
			EffectiveStatus: effectiveStatus,
			Invitee:         invitee,
			InvitedBy:       invitedBy,
		})
	}

	return &ListGroupInvitationsOutput{
		Invitations: details,
		Total:       total,
		TotalPages:  appshared.CalcTotalPages(total, limit),
	}, nil
}
