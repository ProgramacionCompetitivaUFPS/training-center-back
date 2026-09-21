package user

import (
	"context"
	"errors"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
)

func TestListUserFilterOptions_Success(t *testing.T) {
	repo := &mockUserRepository{
		findFilterOptionsFn: func(_ context.Context) (domain.FilterOptions, error) {
			return domain.FilterOptions{
				Countries:    []string{"Colombia", "Mexico"},
				Cities:       []string{"Bogota", "Cucuta"},
				Institutions: []string{"UFPS"},
			}, nil
		},
	}
	uc := NewListUserFilterOptionsUseCase(repo)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Countries) != 2 || result.Countries[0] != "Colombia" {
		t.Errorf("unexpected countries: %+v", result.Countries)
	}
	if len(result.Cities) != 2 {
		t.Errorf("unexpected cities: %+v", result.Cities)
	}
	if len(result.Institutions) != 1 || result.Institutions[0] != "UFPS" {
		t.Errorf("unexpected institutions: %+v", result.Institutions)
	}
}

func TestListUserFilterOptions_RepoError_Propagates(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockUserRepository{
		findFilterOptionsFn: func(_ context.Context) (domain.FilterOptions, error) {
			return domain.FilterOptions{}, repoErr
		},
	}
	uc := NewListUserFilterOptionsUseCase(repo)

	_, err := uc.Execute(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error to propagate, got %v", err)
	}
}
