package problem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestDeleteProblem_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should delete own problem, got: %v", err)
	}
}

func TestDeleteProblem_Success_Admin(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should delete any problem, got: %v", err)
	}
}

func TestDeleteProblem_ProblemInActiveContest(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{inActive: true}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected error when problem is in active contest, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeProblemInActiveContest {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeProblemInActiveContest, appErr.Code)
	}
}

func TestDeleteProblem_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not delete a problem, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestDeleteProblem_ConfirmSlugMismatch(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: "wrong-slug",
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("mismatched confirm slug should return error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeSlugMismatch {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeSlugMismatch, appErr.Code)
	}
}

func TestDeleteProblem_EmptyConfirmSlug(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: "",
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("empty confirm slug should return error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeSlugMismatch {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeSlugMismatch, appErr.Code)
	}
}

func TestDeleteProblem_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestDeleteProblem_RemovesContestAssociationsInTransaction(t *testing.T) {
	// A draft can still belong to a (scheduled) contest: it was unpublished after being added.
	p := newDraftProblem()
	var deleted []string
	repo := repoWith(p)
	repo.deleteFn = func(_ context.Context, id string) error {
		deleted = append(deleted, id)
		return nil
	}
	remover := &mockContestAssociationRemover{}
	tx := &mockTxManager{}
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, remover, tx)

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if tx.calls != 1 {
		t.Errorf("expected 1 transaction, got %d", tx.calls)
	}
	if len(remover.removedIDs) != 1 || remover.removedIDs[0] != p.ID() {
		t.Errorf("expected contest associations removed for %q, got %v", p.ID(), remover.removedIDs)
	}
	if len(deleted) != 1 || deleted[0] != p.ID() {
		t.Errorf("expected problem %q deleted, got %v", p.ID(), deleted)
	}
}

func TestDeleteProblem_ContestRemovalFails_ProblemNotDeleted(t *testing.T) {
	repo := repoWith(newDraftProblem())
	deleteCalled := false
	repo.deleteFn = func(_ context.Context, _ string) error {
		deleteCalled = true
		return nil
	}
	remover := &mockContestAssociationRemover{err: apperror.NewInternal()}
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, remover, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected error when removing contest associations fails, got nil")
	}
	if deleteCalled {
		t.Error("problem must not be deleted when contest association removal fails")
	}
}

func TestDeleteProblem_PublishedIsRejectedEvenForTheAuthorAndAdmin(t *testing.T) {
	for name, user := range map[string]appshared.CurrentUser{"author": asCoach(authorID), "admin": asAdmin("someone-else")} {
		t.Run(name, func(t *testing.T) {
			deleted := false
			repo := repoWith(newPublishedProblem())
			repo.deleteFn = func(context.Context, string) error { deleted = true; return nil }
			uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

			err := uc.Execute(context.Background(), DeleteProblemInput{Slug: testSlug, ConfirmSlug: testSlug, CurrentUser: user})

			ae := appErr(t, err)
			assert.Equal(t, apperror.KindBadRequest, ae.Kind)
			assert.Equal(t, ErrCodeProblemIsPublished, ae.Code)
			assert.False(t, deleted, "nothing may be deleted when the request is refused")
		})
	}
}

func TestDeleteProblem_AfterUnpublishingItCanBeDeleted(t *testing.T) {
	// Unpublished = draft: the extra step is the whole point, not a permanent lock.
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{Slug: testSlug, ConfirmSlug: testSlug, CurrentUser: asCoach(authorID)})
	assert.NoError(t, err)
}
