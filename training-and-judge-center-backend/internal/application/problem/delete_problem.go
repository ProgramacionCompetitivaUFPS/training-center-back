package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteProblemInput struct {
	Slug        string
	ConfirmSlug string
	CurrentUser appshared.CurrentUser
}

type DeleteProblemUseCase struct {
	repo           problem.Repository
	fileStorage    ProblemFileRepository
	contestChecker ActiveContestChecker
	contestRemover ContestAssociationRemover
	txManager      appshared.TransactionManager
}

// maxRetries is the number of retry attempts for storage cleanup operations after
// a problem is deleted from the database. Chosen to tolerate transient failures
// without blocking the response indefinitely.
const maxRetries = 3

func NewDeleteProblemUseCase(
	repo problem.Repository,
	fileStorage ProblemFileRepository,
	contestChecker ActiveContestChecker,
	contestRemover ContestAssociationRemover,
	txManager appshared.TransactionManager,
) *DeleteProblemUseCase {
	return &DeleteProblemUseCase{
		repo:           repo,
		fileStorage:    fileStorage,
		contestChecker: contestChecker,
		contestRemover: contestRemover,
		txManager:      txManager,
	}
}

func (uc *DeleteProblemUseCase) Execute(ctx context.Context, in DeleteProblemInput) error {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if in.ConfirmSlug != p.Slug().String() {
		return apperror.NewBadRequest(problem.ErrCodeSlugMismatch, "Confirmation slug does not match the problem slug")
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.IsAdmin()
	if p.AuthorID() != viewerID && !isAdmin {
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author or Admin can delete this problem")
	}

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Submissions made inside a contest are part of its official result: deleting the problem
		// would silently change the standings of that contest, finished or not. Unpublishing it
		// keeps all of that and still stops new use. Checked inside the transaction, first, because
		// it locks the problem row: a submission being inserted right now commits before the check
		// and no new one can be inserted until the deletion ends.
		hasHistory, err := uc.contestChecker.HasContestSubmissions(txCtx, p.ID())
		if err != nil {
			return err
		}
		if hasHistory {
			return apperror.NewConflict(ErrCodeProblemHasContestHistory, "Cannot delete a problem with submissions made in a contest: it would change that contest's results. Unpublish it instead.")
		}

		inActive, err := uc.contestChecker.IsProblemInActiveContest(txCtx, p.ID())
		if err != nil {
			return err
		}
		if inActive {
			return apperror.NewConflict(problem.ErrCodeProblemInActiveContest, "Cannot delete a problem that is currently being used in an active contest")
		}

		if err := uc.contestRemover.RemoveProblemFromContests(txCtx, p.ID()); err != nil {
			return err
		}
		return uc.repo.Delete(txCtx, p.ID())
	})
	if err != nil {
		return err
	}

	var cleanupFailed bool

	if p.TestCasesKey() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFilesWithPrefix(ctx, *p.TestCasesKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but test cases storage cleanup failed", "prefix", *p.TestCasesKey(), "error", err)
			cleanupFailed = true
		}
	}

	for _, sol := range p.Solutions() {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, sol.FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but solution storage cleanup failed", "key", sol.FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if p.Checker() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, p.Checker().FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but checker storage cleanup failed", "key", p.Checker().FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if p.Validator() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, p.Validator().FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but validator storage cleanup failed", "key", p.Validator().FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if cleanupFailed {
		return apperror.NewInternal()
	}

	return nil
}
