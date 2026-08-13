package problem

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetLatestProblemValidationInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type GetLatestProblemValidationOutput struct {
	// Found is false when this problem has never had a publish attempt —
	// a normal state for a problem that was just created, not an error.
	Found    bool
	Terminal bool
	Passed   bool
	Status   string
	Report   ValidationReport
}

// GetLatestProblemValidationUseCase answers "what happened with the last
// publish attempt for this problem" — for a client that lost track of the
// ValidationID (e.g. closed the tab while AwaitProblemValidationUseCase was
// still polling on the server). It reuses GetProblemValidationStatusUseCase
// once it has resolved slug -> latest ValidationID, instead of duplicating
// the resultJSON decoding.
type GetLatestProblemValidationUseCase struct {
	repo           problem.Repository
	validationRepo problem.ProblemValidationRepository
	statusCheck    *GetProblemValidationStatusUseCase
}

func NewGetLatestProblemValidationUseCase(
	repo problem.Repository,
	validationRepo problem.ProblemValidationRepository,
	statusCheck *GetProblemValidationStatusUseCase,
) *GetLatestProblemValidationUseCase {
	return &GetLatestProblemValidationUseCase{repo: repo, validationRepo: validationRepo, statusCheck: statusCheck}
}

func (uc *GetLatestProblemValidationUseCase) Execute(ctx context.Context, in GetLatestProblemValidationInput) (*GetLatestProblemValidationOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !p.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can view this problem's validation status")
	}

	existing, found, err := uc.validationRepo.FindLatestByProblemID(ctx, p.ID())
	if err != nil {
		return nil, err
	}
	if !found {
		return &GetLatestProblemValidationOutput{Found: false}, nil
	}

	status, err := uc.statusCheck.Execute(ctx, GetProblemValidationStatusInput{ValidationID: existing.ID()})
	if err != nil {
		return nil, err
	}
	return &GetLatestProblemValidationOutput{
		Found:    true,
		Terminal: status.Terminal,
		Passed:   status.Passed,
		Status:   status.Status,
		Report:   status.Report,
	}, nil
}
