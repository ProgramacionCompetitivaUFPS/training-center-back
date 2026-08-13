package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newPublishUseCase(repo *mockProblemRepository, validationRepo *mockValidationRepository, queue *mockValidationQueue) *PublishProblemUseCase {
	return NewPublishProblemUseCase(repo, validationRepo, queue)
}

func TestExecute_MissingFields_ReturnsResolvedOutput(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{}
	queue := &mockValidationQueue{}
	uc := newPublishUseCase(repo, validationRepo, queue)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.MissingFields) == 0 {
		t.Fatal("expected missing fields, got none")
	}
	if out.ValidationID != "" {
		t.Errorf("ValidationID: got %q, want empty", out.ValidationID)
	}
	if len(queue.published) != 0 {
		t.Error("expected the queue to never be used when fields are missing")
	}
}

func TestExecute_AlreadyPublished_ReturnsConflict(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := newPublishUseCase(repo, &mockValidationRepository{}, &mockValidationQueue{})

	_, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	var appErr *apperror.AppError
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeAlreadyPublished {
		t.Errorf("expected ErrCodeAlreadyPublished, got %v", err)
	}
}

func TestExecute_Forbidden_ReturnsForbidden(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := newPublishUseCase(repo, &mockValidationRepository{}, &mockValidationQueue{})

	_, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asContestant(strangerID)})
	var appErr *apperror.AppError
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected Forbidden, got %v", err)
	}
}

func TestExecute_CompleteProblem_NoActiveValidation_EnqueuesAndReturnsValidationID(t *testing.T) {
	repo := repoWith(newCompleteDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
	}
	queue := &mockValidationQueue{}
	uc := newPublishUseCase(repo, validationRepo, queue)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ValidationID == "" {
		t.Error("expected a ValidationID, got empty")
	}
	if len(out.MissingFields) != 0 {
		t.Errorf("MissingFields: got %v, want none", out.MissingFields)
	}
	if len(queue.published) != 1 {
		t.Fatalf("expected exactly one message published, got %d", len(queue.published))
	}
	if queue.published[0].ValidationID != out.ValidationID {
		t.Errorf("published ValidationID: got %q, want %q", queue.published[0].ValidationID, out.ValidationID)
	}
	if queue.published[0].Priority != QueuePriorityPublishValidation {
		t.Errorf("Priority: got %d, want %d", queue.published[0].Priority, QueuePriorityPublishValidation)
	}
	if queue.published[0].Slug != testSlug {
		t.Errorf("published Slug: got %q, want %q", queue.published[0].Slug, testSlug)
	}
}

func TestExecute_ActiveValidationExists_ReusesIt(t *testing.T) {
	repo := repoWith(newCompleteDraftProblem())
	existing, err := domainProblem.NewProblemValidation("existing-id", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return existing, true, nil
		},
		saveFn: func(_ context.Context, _ *domainProblem.ProblemValidation) error {
			t.Fatal("Save should not be called when an active validation already exists")
			return nil
		},
	}
	queue := &mockValidationQueue{}
	uc := newPublishUseCase(repo, validationRepo, queue)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ValidationID != "existing-id" {
		t.Errorf("ValidationID: got %q, want existing-id", out.ValidationID)
	}
	if len(queue.published) != 0 {
		t.Error("expected the queue to never be used when reusing an existing validation")
	}
}

func TestExecute_TerminalValidationExists_EnqueuesNewOne(t *testing.T) {
	repo := repoWith(newCompleteDraftProblem())
	terminal, err := domainProblem.NewProblemValidation("old-id", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if err := terminal.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}
	if err := terminal.MarkFailed(`{"missingFields":["statement"]}`, testNow); err != nil {
		t.Fatalf("fixture MarkFailed: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return terminal, true, nil
		},
	}
	queue := &mockValidationQueue{}
	uc := newPublishUseCase(repo, validationRepo, queue)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ValidationID == "" || out.ValidationID == "old-id" {
		t.Errorf("expected a fresh ValidationID, got %q", out.ValidationID)
	}
	if len(queue.published) != 1 {
		t.Errorf("expected exactly one message published, got %d", len(queue.published))
	}
}

func TestExecute_SaveRaceCondition_ReusesWinner(t *testing.T) {
	repo := repoWith(newCompleteDraftProblem())
	winner, err := domainProblem.NewProblemValidation("winner-id", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	calls := 0
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			calls++
			if calls == 1 {
				return nil, false, nil // first check: no active validation yet
			}
			return winner, true, nil // race-retry lookup: someone else already won
		},
		saveFn: func(_ context.Context, _ *domainProblem.ProblemValidation) error {
			return apperror.NewConflict(domainProblem.ErrCodeValidationInProgress, "a validation is already in progress for this problem")
		},
	}
	queue := &mockValidationQueue{}
	uc := newPublishUseCase(repo, validationRepo, queue)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ValidationID != "winner-id" {
		t.Errorf("ValidationID: got %q, want winner-id", out.ValidationID)
	}
	if len(queue.published) != 0 {
		t.Error("expected the queue to never be used when losing the insert race")
	}
}

func TestExecute_QueuePublishFails_ReturnsError(t *testing.T) {
	repo := repoWith(newCompleteDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
	}
	queue := &mockValidationQueue{
		publishFn: func(_ context.Context, _ ValidationQueueMessage) error {
			return apperror.NewInternal()
		},
	}
	uc := newPublishUseCase(repo, validationRepo, queue)

	_, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err == nil {
		t.Error("expected error when the queue publish fails, got nil")
	}
}
