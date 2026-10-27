package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRequestValidationUseCase(repo *mockProblemRepository, validationRepo *mockValidationRepository, queue *mockValidationQueue) *RequestProblemValidationUseCase {
	return NewRequestProblemValidationUseCase(repo, validationRepo, queue)
}

func TestRequestValidation_CompleteDraft_OpensACheckOnlyTicket(t *testing.T) {
	var saved *domainProblem.ProblemValidation
	validationRepo := &mockValidationRepository{
		saveFn: func(_ context.Context, v *domainProblem.ProblemValidation) error {
			saved = v
			return nil
		},
	}
	queue := &mockValidationQueue{}
	uc := newRequestValidationUseCase(repoWith(newCompleteDraftProblem()), validationRepo, queue)

	out, err := uc.Execute(context.Background(), RequestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil || saved.ID() != out.ValidationID {
		t.Fatalf("expected the ticket %q to be saved, got %+v", out.ValidationID, saved)
	}
	if saved.PublishesOnPass() {
		t.Error("a validation requested with Validar must not publish the problem when it passes")
	}
	if len(queue.published) != 1 || queue.published[0].ValidationID != out.ValidationID {
		t.Errorf("expected the ticket to be queued once, got %+v", queue.published)
	}
}

func TestRequestValidation_MissingFields_ReturnsThemWithoutQueueing(t *testing.T) {
	queue := &mockValidationQueue{}
	uc := newRequestValidationUseCase(repoWith(newDraftProblem()), &mockValidationRepository{}, queue)

	out, err := uc.Execute(context.Background(), RequestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.MissingFields) == 0 || out.ValidationID != "" {
		t.Errorf("expected missing fields and no ticket, got %+v", out)
	}
	if len(queue.published) != 0 {
		t.Error("nothing should be queued when fields are missing")
	}
}

func TestRequestValidation_AlreadyPublished_ReturnsConflict(t *testing.T) {
	uc := newRequestValidationUseCase(repoWith(newPublishedProblem()), &mockValidationRepository{}, &mockValidationQueue{})

	_, err := uc.Execute(context.Background(), RequestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeAlreadyPublished {
		t.Errorf("expected ErrCodeAlreadyPublished, got %v", err)
	}
}

func TestRequestValidation_Forbidden_ReturnsForbidden(t *testing.T) {
	uc := newRequestValidationUseCase(repoWith(newCompleteDraftProblem()), &mockValidationRepository{}, &mockValidationQueue{})

	_, err := uc.Execute(context.Background(), RequestProblemValidationInput{Slug: testSlug, CurrentUser: asContestant(strangerID)})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected ErrCodeInsufficientPermissions, got %v", err)
	}
}

func TestRequestValidation_ActiveValidationExists_ReusesIt(t *testing.T) {
	existing, err := domainProblem.NewProblemValidation("existing-id", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return existing, true, nil
		},
	}
	queue := &mockValidationQueue{}
	uc := newRequestValidationUseCase(repoWith(newCompleteDraftProblem()), validationRepo, queue)

	out, err := uc.Execute(context.Background(), RequestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ValidationID != "existing-id" || len(queue.published) != 0 {
		t.Errorf("expected to reuse existing-id without queueing, got %q and %d queued", out.ValidationID, len(queue.published))
	}
}

// ── Publish after a Validar ──────────────────────────────────────────────────

func passedCheckOnly(id string, requestedAt time.Time) *domainProblem.ProblemValidation {
	v, err := domainProblem.NewCheckOnlyValidation(id, testProbID, shared.RestoreUserID(authorID), requestedAt)
	if err != nil {
		panic(err)
	}
	if err := v.Start(requestedAt); err != nil {
		panic(err)
	}
	if err := v.MarkPassed(`{"validationLogs":["ok"]}`, requestedAt.Add(time.Minute)); err != nil {
		panic(err)
	}
	return v
}

func latestIs(v *domainProblem.ProblemValidation) *mockValidationRepository {
	return &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return v, true, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
}

func TestPublish_AfterAPassingCheckThatIsStillCurrent_PublishesWithoutValidatingAgain(t *testing.T) {
	check := passedCheckOnly("check-id", testNow) // the problem was last updated at testNow: nothing changed since
	publisher := &mockProblemPublisher{}
	queue := &mockValidationQueue{}
	uc := NewPublishProblemUseCase(repoWith(newCompleteDraftProblem()), latestIs(check), queue, publisher)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher.calls != 1 {
		t.Errorf("expected the problem to be marked published once, got %d", publisher.calls)
	}
	if len(queue.published) != 0 {
		t.Error("a current passing check needs no second validation")
	}
	if out.ValidationID != "check-id" {
		t.Errorf("ValidationID: got %q, want the check that justified publishing", out.ValidationID)
	}
}

func TestPublish_AfterAPassingCheckThatIsStale_ValidatesAgain(t *testing.T) {
	check := passedCheckOnly("check-id", testNow.Add(-time.Hour)) // the problem changed (testNow) after the check
	publisher := &mockProblemPublisher{}
	queue := &mockValidationQueue{}
	uc := NewPublishProblemUseCase(repoWith(newCompleteDraftProblem()), latestIs(check), queue, publisher)

	out, err := uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher.calls != 0 {
		t.Error("a stale check must not publish the problem")
	}
	if len(queue.published) != 1 || out.ValidationID == "check-id" {
		t.Errorf("expected a fresh validation to be queued, got %d queued and ValidationID %q", len(queue.published), out.ValidationID)
	}
}

func TestPublish_WhileACheckIsRunning_ReturnsConflict(t *testing.T) {
	running, err := domainProblem.NewCheckOnlyValidation("check-id", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	publisher := &mockProblemPublisher{}
	uc := NewPublishProblemUseCase(repoWith(newCompleteDraftProblem()), latestIs(running), &mockValidationQueue{}, publisher)

	_, err = uc.Execute(context.Background(), PublishProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
	if publisher.calls != 0 {
		t.Error("nothing should be published while the check runs")
	}
}
