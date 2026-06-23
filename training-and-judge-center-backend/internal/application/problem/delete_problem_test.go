package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestDeleteProblem_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

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
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

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
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{inActive: true})

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
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

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
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

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
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: "", // empty
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("empty confirm slug should return validation error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestDeleteProblem_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	uc := NewDeleteProblemUseCase(repo, &mockFileStorage{}, &mockActiveContestChecker{})

	err := uc.Execute(context.Background(), DeleteProblemInput{
		Slug:        testSlug,
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}
