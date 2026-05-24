package material

import (
	"context"
	"log/slog"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func checkGroupAccess(ctx context.Context, mp GroupMemberProvider, user appshared.CurrentUser, groupID string, visibility GroupVisibility) error {
	if visibility == GroupVisibilityVisible || user.IsAdmin() {
		return nil
	}
	if visibility != GroupVisibilityNotVisible {
		slog.WarnContext(ctx, "unrecognised group visibility value, treating as NOT_VISIBLE", "visibility", visibility, "group_id", groupID)
	}
	isMember, err := mp.IsMemberOfGroup(ctx, user.ID, groupID)
	if err != nil {
		return err
	}
	if !isMember {
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "you do not have permission to view materials in this group")
	}
	return nil
}
