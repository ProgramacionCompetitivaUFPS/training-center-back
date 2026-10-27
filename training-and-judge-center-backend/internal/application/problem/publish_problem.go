package problem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type PublishProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type PublishProblemOutput struct {
	Slug           string
	MissingFields  []string
	ValidationLogs []string
	ValidationID   string
}

type PublishProblemUseCase struct {
	repo           problem.Repository
	validationRepo problem.ProblemValidationRepository
	queue          ValidationQueue
	publisher      ProblemPublisher
}

func NewPublishProblemUseCase(repo problem.Repository, validationRepo problem.ProblemValidationRepository, queue ValidationQueue, publisher ProblemPublisher) *PublishProblemUseCase {
	return &PublishProblemUseCase{repo: repo, validationRepo: validationRepo, queue: queue, publisher: publisher}
}

func (uc *PublishProblemUseCase) Execute(ctx context.Context, in PublishProblemInput) (*PublishProblemOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	if !p.CanBeEditedBy(viewerID, in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can publish this problem")
	}

	if p.Status().IsPublished() {
		return nil, apperror.NewConflict(problem.ErrCodeAlreadyPublished, "Problem is already published")
	}

	missing, logs := requiredFieldsForPublish(p)
	if len(missing) > 0 {
		return &PublishProblemOutput{Slug: p.Slug().String(), MissingFields: missing, ValidationLogs: logs}, nil
	}

	latest, found, err := uc.validationRepo.FindLatestByProblemID(ctx, p.ID())
	if err != nil {
		return nil, err
	}
	if found {
		switch {
		case !latest.Status().IsFinal() && latest.PublishesOnPass():
			// Reuse an in-flight validation instead of enqueueing a duplicate.
			return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: latest.ID()}, nil
		case !latest.Status().IsFinal():
			// A "Validar" run does not publish when it passes, so waiting on it would report a
			// success for a problem that stays a draft.
			return nil, problemValidationInProgress()
		case latest.Status().IsPassed() && !latest.IsJudgingUpdate() && !p.UpdatedAt().After(latest.RequestedAt()):
			// The draft was validated and has not changed since: publishing it needs no second run.
			// The check is repeated inside the UPDATE: an edit saved after the read above must not
			// be published on the strength of this validation, so then it validates again.
			published, err := uc.publisher.MarkPublishedIfUnchanged(ctx, p.ID(), p.UpdatedAt(), time.Now())
			if err != nil {
				return nil, err
			}
			if published {
				return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: latest.ID()}, nil
			}
		}
	}

	id, err := enqueueValidation(ctx, uc.validationRepo, uc.queue, p, viewerID, true)
	if err != nil {
		return nil, err
	}
	return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: id}, nil
}

// problemValidationInProgress is what a publish gets while a "Validar" run, which would not publish, holds the problem.
func problemValidationInProgress() error {
	return apperror.NewConflict(problem.ErrCodeValidationInProgress, "A validation is already running for this problem; wait for it to finish and publish again")
}

// enqueueValidation opens a validation ticket for the draft and queues it for the worker.
// publishOnPass tells the worker whether to publish the problem when the ticket passes.
// If another request opened a ticket in the meantime, that one is returned instead.
func enqueueValidation(ctx context.Context, validationRepo problem.ProblemValidationRepository, queue ValidationQueue, p *problem.Problem, requestedBy shared.UserID, publishOnPass bool) (string, error) {
	now := time.Now()
	newTicket := problem.NewCheckOnlyValidation
	if publishOnPass {
		newTicket = problem.NewProblemValidation
	}
	v, err := newTicket(uuid.New().String(), p.ID(), requestedBy, now)
	if err != nil {
		return "", err
	}
	if err := validationRepo.Save(ctx, v); err != nil {
		if isValidationInProgress(err) {
			if existing, found, ferr := validationRepo.FindLatestByProblemID(ctx, p.ID()); ferr == nil && found {
				if publishOnPass && !existing.PublishesOnPass() {
					// Lost the race to a "Validar": that run will not publish, so a publish must not wait on it.
					return "", problemValidationInProgress()
				}
				return existing.ID(), nil
			}
		}
		return "", err
	}

	if err := queue.Publish(ctx, ValidationQueueMessage{
		ValidationID: v.ID(),
		ProblemID:    p.ID(),
		Slug:         p.Slug().String(),
		Priority:     QueuePriorityPublishValidation,
		EnqueuedAt:   now,
	}); err != nil {
		return "", err
	}
	return v.ID(), nil
}

func isValidationInProgress(err error) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Code == problem.ErrCodeValidationInProgress
}

func requiredFieldsForPublish(p *problem.Problem) (missing []string, logs []string) {
	logs = append(logs, fmt.Sprintf("✓ Title: %s", p.Title().String()))

	if p.Statement().Value() == nil {
		missing = append(missing, "statement")
		logs = append(logs, "✗ Statement: Missing (required)")
	} else {
		logs = append(logs, "✓ Statement: present")
	}

	if p.TimeLimit() == nil {
		missing = append(missing, "timeLimit")
		logs = append(logs, "✗ Time limit: Missing (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Time limit: %dms", p.TimeLimit().Milliseconds()))
	}

	if p.MemoryLimit() == nil {
		missing = append(missing, "memoryLimit")
		logs = append(logs, "✗ Memory limit: Missing (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Memory limit: %d MiB", p.MemoryLimit().Megabytes()))
	}

	if p.TestCasesKey() == nil {
		missing = append(missing, "testCases")
		logs = append(logs, "✗ Test cases: Not uploaded (required)")
	} else {
		logs = append(logs, "✓ Test cases: uploaded")
	}

	if len(p.Solutions()) == 0 {
		missing = append(missing, "solution")
		logs = append(logs, "✗ Solution: Not uploaded (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Solution: %d uploaded", len(p.Solutions())))
	}

	return missing, logs
}
