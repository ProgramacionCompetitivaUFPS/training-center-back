package user

import (
	"context"
	"strings"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListUsersInput struct {
	Roles       []string
	Status      string
	Country     string
	City        string
	Institution string
	SearchField string
	SearchTerm  string
	Sort        string
	Order       string
	Page        int
	Limit       int
}

type ListUsersOutput struct {
	Users      []*user.User
	TotalCount int
	Page       int
	Limit      int
}

type ListUsersUseCase struct {
	repo user.UserRepository
}

func NewListUsersUseCase(repo user.UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{repo: repo}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, input ListUsersInput) (*ListUsersOutput, error) {
	var filter user.UserFilter
	var fieldErrors []apperror.FieldError

	// Validate and parse roles
	for _, r := range input.Roles {
		role, err := user.NewRole(strings.TrimSpace(r))
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{
				Field:   "role",
				Message: "Role must be one of: ADMIN, COACH, CONTESTANT",
			})
			break
		}
		filter.Roles = append(filter.Roles, role)
	}

	// Validate status
	if input.Status != "" {
		s, err := user.NewStatus(input.Status)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{
				Field:   "status",
				Message: "Status must be ACTIVE or DEACTIVATED",
			})
		} else {
			filter.Status = &s
		}
	}

	// Validate sort
	sort := input.Sort
	if sort == "" {
		sort = "createdAt"
	}
	if !user.ValidSortFields[sort] {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "sort",
			Message: "Sort field must be one of: createdAt, name, nickname, email, deactivatedAt",
		})
	}

	// Validate order
	order := strings.ToLower(input.Order)
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "order",
			Message: "Sort order must be: asc or desc",
		})
	}

	// Validate searchField
	searchField := input.SearchField
	if searchField == "" {
		searchField = "all"
	}
	if !user.ValidSearchFields[searchField] {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "searchField",
			Message: "Search field must be one of: name, nickname, email, institution, all",
		})
	}

	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	// Apply defaults and caps
	page := input.Page
	if page < 1 {
		page = 1
	}
	limit := input.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter.Country = input.Country
	filter.City = input.City
	filter.Institution = input.Institution
	filter.SearchField = searchField
	filter.SearchTerm = input.SearchTerm
	filter.Sort = sort
	filter.Order = order
	filter.Page = page
	filter.Limit = limit

	users, total, err := uc.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, apperror.NewInternal()
	}

	return &ListUsersOutput{
		Users:      users,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
	}, nil
}
