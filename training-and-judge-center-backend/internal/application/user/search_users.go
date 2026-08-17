package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	minSearchQueryLength = 2
	defaultSearchLimit   = 10
	maxSearchLimit       = 20
)

type SearchUsersInput struct {
	Query string
	Limit int
}

type SearchUsersOutput struct {
	Users []UserSearchResult
}

// UserSearchResult is the minimal, non-sensitive projection returned by
// autocomplete search — deliberately excludes email/country/city/institution.
type UserSearchResult struct {
	ID       string
	Nickname string
	Name     string
}

type SearchUsersUseCase struct {
	repo user.Repository
}

func NewSearchUsersUseCase(repo user.Repository) *SearchUsersUseCase {
	return &SearchUsersUseCase{repo: repo}
}

// Execute finds active users whose name or nickname partially matches the
// query. Intended for authenticated Coach/Admin autocomplete flows (problem
// modifiers, group invites, contest problem pickers) — deliberately matches
// only name/nickname (never email/institution, unlike the admin user list's
// SearchByAll) so the endpoint can't be used to confirm whether a given email
// belongs to an account.
func (uc *SearchUsersUseCase) Execute(ctx context.Context, in SearchUsersInput) (*SearchUsersOutput, error) {
	if len([]rune(in.Query)) < minSearchQueryLength {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "q", Message: "Search query must be at least 2 characters"},
		})
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	users, err := uc.repo.SearchActive(ctx, in.Query, limit)
	if err != nil {
		return nil, err
	}

	results := make([]UserSearchResult, len(users))
	for i, u := range users {
		results[i] = UserSearchResult{
			ID:       u.ID(),
			Nickname: u.Nickname().String(),
			Name:     u.Name(),
		}
	}

	return &SearchUsersOutput{Users: results}, nil
}
