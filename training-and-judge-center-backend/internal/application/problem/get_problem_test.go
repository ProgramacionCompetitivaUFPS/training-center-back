package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetProblem_AuthorCanSeeDraft(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newDraftProblem()), &mockUserProvider{}, &mockFileStorage{})

	out, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should see own draft, got error: %v", err)
	}
	if out.Problem.Slug != testSlug {
		t.Errorf("unexpected slug: %q", out.Problem.Slug)
	}
	if out.Modifiers == nil {
		t.Error("author should receive modifiers list")
	}
}

func TestGetProblem_AdminCanSeeDraft(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newDraftProblem()), &mockUserProvider{}, &mockFileStorage{})

	_, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should see any draft, got error: %v", err)
	}
}

func TestGetProblem_ModifierCanSeeDraft(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newDraftProblemWithModifier()), &mockUserProvider{}, &mockFileStorage{})

	_, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(modifierID),
	})
	if err != nil {
		t.Fatalf("modifier should see draft, got error: %v", err)
	}
}

func TestGetProblem_StrangerCannotSeeDraft(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newDraftProblem()), &mockUserProvider{}, &mockFileStorage{})

	_, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not see draft, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestGetProblem_AnyoneCanSeePublished(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newPublishedProblem()), &mockUserProvider{}, &mockFileStorage{})

	_, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err != nil {
		t.Fatalf("anyone should see published problem, got error: %v", err)
	}
}

func TestGetProblem_NonEditorDoesNotSeeModifiersOrFiles(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newPublishedProblem()), &mockUserProvider{}, &mockFileStorage{})

	out, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Modifiers != nil {
		t.Error("non-editor should not receive modifiers list")
	}
	if out.Files != nil {
		t.Error("non-editor should not receive files availability")
	}
}

func TestGetProblem_IncludesSamplesForAnyViewer(t *testing.T) {
	storage := &mockFileStorage{
		listFilesFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{
				"problems/test-problem/testcases/xyz/data/sample/1.in",
				"problems/test-problem/testcases/xyz/data/sample/1.ans",
			}, nil
		},
		downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
			return []byte(path), nil
		},
	}
	uc := NewGetProblemUseCase(repoWith(newPublishedProblemWithTestCases()), &mockUserProvider{}, storage)

	out, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Samples) != 1 {
		t.Fatalf("expected 1 sample for any viewer of a published problem, got %d", len(out.Samples))
	}
}

func TestGetProblem_NoTestCasesReturnsEmptySamplesNotError(t *testing.T) {
	uc := NewGetProblemUseCase(repoWith(newPublishedProblem()), &mockUserProvider{}, &mockFileStorage{})

	out, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Samples == nil || len(out.Samples) != 0 {
		t.Errorf("expected empty (non-nil) samples slice, got %v", out.Samples)
	}
}

func TestGetProblem_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	uc := NewGetProblemUseCase(repo, &mockUserProvider{}, &mockFileStorage{})

	_, err := uc.Execute(context.Background(), GetProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeNotFound {
		t.Errorf("expected NOT_FOUND, got %q", appErr.Code)
	}
}
