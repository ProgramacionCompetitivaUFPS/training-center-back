package submission

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateSubmissionVisibilityInput struct {
	CurrentUser   appshared.CurrentUser
	SubmissionID  string
	NewVisibility string
}

type UpdateSubmissionVisibilityUseCase struct {
	submissionRepo domainSubmission.Repository
}

func NewUpdateSubmissionVisibilityUseCase(submissionRepo domainSubmission.Repository) *UpdateSubmissionVisibilityUseCase {
	return &UpdateSubmissionVisibilityUseCase{submissionRepo: submissionRepo}
}

func (uc *UpdateSubmissionVisibilityUseCase) Execute(ctx context.Context, in UpdateSubmissionVisibilityInput) error {
	sub, err := uc.submissionRepo.FindByID(ctx, in.SubmissionID)
	if err != nil {
		return err
	}

	if in.CurrentUser.ID != sub.UserID().String() {
		return apperror.NewForbidden(domainSubmission.ErrCodeAccessDenied, "only the author can change visibility")
	}

	newVis, err := domainSubmission.NewVisibility(in.NewVisibility)
	if err != nil {
		return err
	}

	sub.SetVisibility(newVis)

	return uc.submissionRepo.UpdateVisibility(ctx, sub)
}
