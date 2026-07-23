package group

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteGroupInput struct {
	GroupID          string
	ConfirmationName string
	CurrentUser      appshared.CurrentUser
}

type DeleteGroupOutput struct {
	GroupID             string
	GroupName           string
	ContestsDeleted     int
	MaterialsDeleted    int
	StandingsDeleted    int
	SubmissionsOrphaned int
	MembersRemoved      int
}

type DeleteGroupUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	deletionProvider GroupDeletionProvider
	standingsCache   GroupStandingsInvalidator
	txManager        appshared.TransactionManager
}

func NewDeleteGroupUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	deletionProvider GroupDeletionProvider,
	standingsCache GroupStandingsInvalidator,
	txManager appshared.TransactionManager,
) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		deletionProvider: deletionProvider,
		standingsCache:   standingsCache,
		txManager:        txManager,
	}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, in DeleteGroupInput) (*DeleteGroupOutput, error) {
	g, err := uc.groupRepo.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	if !g.CanBeDeleted() {
		return nil, apperror.NewForbidden(domainGroup.ErrCodeCannotDeleteGlobalGroup, "the global group cannot be deleted")
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, in.GroupID, in.CurrentUser); err != nil {
		return nil, err
	}

	if in.ConfirmationName == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "confirmationName", Message: "confirmation name is required to delete a group"},
		})
	}
	if in.ConfirmationName != g.Name().Value() {
		return nil, apperror.NewBadRequest(ErrCodeConfirmationMismatch, "confirmation name does not match the group name")
	}

	counts, err := uc.deletionProvider.GetDeletionCounts(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return uc.groupRepo.Delete(txCtx, in.GroupID)
	}); err != nil {
		return nil, err
	}

	// Use a detached context so cache invalidation is not skipped if the
	// client disconnected while the transaction was running.
	invalidateCtx := context.WithoutCancel(ctx)
	for _, contestID := range counts.ContestIDs {
		_ = uc.standingsCache.Invalidate(invalidateCtx, contestID)
	}

	return &DeleteGroupOutput{
		GroupID:             g.ID(),
		GroupName:           g.Name().Value(),
		ContestsDeleted:     len(counts.ContestIDs),
		MaterialsDeleted:    counts.MaterialsCount,
		StandingsDeleted:    len(counts.ContestIDs),
		SubmissionsOrphaned: counts.SubmissionsCount,
		MembersRemoved:      counts.MembersCount,
	}, nil
}
