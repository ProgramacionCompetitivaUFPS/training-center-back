package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnpublishProblem_Success_Author(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewUnpublishProblemUseCase(repo)

	out, err := uc.Execute(context.Background(), UnpublishProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should unpublish own problem, got: %v", err)
	}
	if out.Problem.Status != "DRAFT" {
		t.Errorf("expected DRAFT after unpublish, got %q", out.Problem.Status)
	}
}

func TestUnpublishProblem_Success_Admin(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewUnpublishProblemUseCase(repo)

	_, err := uc.Execute(context.Background(), UnpublishProblemInput{
		Slug:        testSlug,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should unpublish any problem, got: %v", err)
	}
}

func TestUnpublishProblem_Success_Modifier(t *testing.T) {
	repo := repoWith(newPublishedProblemWithModifier())
	uc := NewUnpublishProblemUseCase(repo)

	_, err := uc.Execute(context.Background(), UnpublishProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(modifierID),
	})
	if err != nil {
		t.Fatalf("modifier should unpublish problem, got: %v", err)
	}
}

func TestUnpublishProblem_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewUnpublishProblemUseCase(repo)

	_, err := uc.Execute(context.Background(), UnpublishProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not unpublish, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeForbidden {
		t.Errorf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestUnpublishProblem_AlreadyDraft(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUnpublishProblemUseCase(repo)

	_, err := uc.Execute(context.Background(), UnpublishProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("unpublishing a draft should return error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeAlreadyDraft {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeAlreadyDraft, appErr.Code)
	}
}
