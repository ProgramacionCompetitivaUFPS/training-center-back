package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

func TestListProblems_PassesSearchToRepository(t *testing.T) {
	var capturedFilters domainProblem.ListFilters
	repo := &mockProblemRepository{
		listFn: func(_ context.Context, f domainProblem.ListFilters) ([]*domainProblem.Problem, int, error) {
			capturedFilters = f
			return nil, 0, nil
		},
	}
	uc := NewListProblemsUseCase(repo, &mockUserProvider{})

	_, err := uc.Execute(context.Background(), ListProblemsInput{
		CurrentUser: asCoach(authorID),
		Search:      "dynamic programming",
		Page:        1,
		Limit:       20,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.Search != "dynamic programming" {
		t.Errorf("expected search filter %q, got %q", "dynamic programming", capturedFilters.Search)
	}
}
