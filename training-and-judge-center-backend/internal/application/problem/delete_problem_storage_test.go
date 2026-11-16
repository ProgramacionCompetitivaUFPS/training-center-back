package problem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func allOwnedFileKeys() []string {
	return []string{
		ownedSolutionKey, ownedAltSolutionKey,
		liveCheckerKey, liveCheckerCompKey,
		ownedValidatorKey, ownedValidatorCompKey,
	}
}

func TestDeleteProblem_DropsEverythingTheProblemOwns_IncludingTheCompiledBinaries(t *testing.T) {
	storage := newRecordingStorage()
	uc := NewDeleteProblemUseCase(repoWith(newDraftProblemOwningFiles()), &mockValidationRepository{}, storage, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{Slug: testSlug, ConfirmSlug: testSlug, CurrentUser: asCoach(authorID)})

	assert.NoError(t, err)
	assert.Equal(t, []string{liveTestCasesKey}, storage.deletedPrefixes)
	assert.ElementsMatch(t, allOwnedFileKeys(), storage.deletedFiles)
}

func TestDeleteProblem_ValidationInProgress_IsRefusedAndNothingIsDeleted(t *testing.T) {
	storage := newRecordingStorage()
	deleted := false
	repo := repoWith(newDraftProblemOwningFiles())
	repo.deleteFn = func(context.Context, string) error { deleted = true; return nil }
	validations := &mockValidationRepository{
		findLatestByProblemIDFn: func(context.Context, string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	removed := &mockContestAssociationRemover{}
	uc := NewDeleteProblemUseCase(repo, validations, storage, &mockActiveContestChecker{}, removed, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{Slug: testSlug, ConfirmSlug: testSlug, CurrentUser: asCoach(authorID)})

	ae := appErr(t, err)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, domainProblem.ErrCodeValidationInProgress, ae.Code)
	assert.False(t, deleted, "the problem must survive while a validation is running")
	assert.Empty(t, removed.removedIDs)
	assert.Zero(t, storage.attempts, "no file may be touched when the delete is refused")
}

func TestDeleteProblem_StorageFails_TriesEveryKeyAndReportsTheError(t *testing.T) {
	storage := newRecordingStorage()
	storage.failEveryDeletion()
	deleted := false
	repo := repoWith(newDraftProblemOwningFiles())
	repo.deleteFn = func(context.Context, string) error { deleted = true; return nil }
	uc := NewDeleteProblemUseCase(repo, &mockValidationRepository{}, storage, &mockActiveContestChecker{}, &mockContestAssociationRemover{}, &mockTxManager{})

	err := uc.Execute(context.Background(), DeleteProblemInput{Slug: testSlug, ConfirmSlug: testSlug, CurrentUser: asCoach(authorID)})

	assert.Equal(t, apperror.KindInternal, appErr(t, err).Kind)
	assert.True(t, deleted, "the problem row is already gone when the storage cleanup runs")
	keys := 1 + len(allOwnedFileKeys())
	assert.Equal(t, keys*maxRetries, storage.attempts, "every key gets its retries; one failure must not stop the rest")
}
