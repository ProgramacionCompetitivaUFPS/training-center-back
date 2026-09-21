package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type ListUserFilterOptionsOutput struct {
	Countries    []string
	Cities       []string
	Institutions []string
}

type ListUserFilterOptionsUseCase struct {
	repo user.Repository
}

func NewListUserFilterOptionsUseCase(repo user.Repository) *ListUserFilterOptionsUseCase {
	return &ListUserFilterOptionsUseCase{repo: repo}
}

// Execute returns the distinct country/city/institution values currently in
// use across all users, so an admin filter can offer a closed selector
// instead of free text with an unknown exact spelling.
func (uc *ListUserFilterOptionsUseCase) Execute(ctx context.Context) (*ListUserFilterOptionsOutput, error) {
	options, err := uc.repo.FindFilterOptions(ctx)
	if err != nil {
		return nil, err
	}

	return &ListUserFilterOptionsOutput{
		Countries:    options.Countries,
		Cities:       options.Cities,
		Institutions: options.Institutions,
	}, nil
}
