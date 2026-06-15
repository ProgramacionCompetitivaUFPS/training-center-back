package group

import (
	"context"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func requireNotContestantForLead(role domainGroup.MemberRole, target *UserDisplay) error {
	if role == domainGroup.MemberRoleLead && target.SystemRole == shared.RoleContestant.String() {
		return apperror.NewBadRequest(domainGroup.ErrCodeInvalidLeadAssignment, "only Coaches and Admins can be assigned as leads")
	}
	return nil
}

func requireLeadOrAdmin(ctx context.Context, memberRepo domainGroup.MemberRepository, groupID string, caller appshared.CurrentUser) error {
	if caller.IsAdmin() {
		return nil
	}
	member, err := memberRepo.FindByGroupAndUser(ctx, groupID, shared.RestoreUserID(caller.ID))
	if err != nil {
		return err
	}
	if member == nil || !member.IsLead() {
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "only leads can manage join requests")
	}
	return nil
}
