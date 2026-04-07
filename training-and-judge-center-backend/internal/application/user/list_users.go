package user

import (
	"context"
	"log/slog"
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
	Users      []UserDTO
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

	sortRaw := input.Sort
	if sortRaw == "" {
		sortRaw = string(user.SortByCreatedAt)
	}
	sortField, err := user.NewSortField(sortRaw)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "sort",
			Message: "Sort field must be one of: createdAt, name, nickname, email, deactivatedAt",
		})
	}

	orderRaw := input.Order
	if orderRaw == "" {
		orderRaw = string(user.SortOrderDesc)
	}
	sortOrder, err := user.NewSortOrder(orderRaw)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "order",
			Message: "Sort order must be: asc or desc",
		})
	}

	searchFieldRaw := input.SearchField
	if searchFieldRaw == "" {
		searchFieldRaw = string(user.SearchByAll)
	}
	searchField, err := user.NewSearchField(searchFieldRaw)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{
			Field:   "searchField",
			Message: "Search field must be one of: name, nickname, email, institution, all",
		})
	}

	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

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
	filter.Sort = sortField
	filter.Order = sortOrder
	filter.Page = page
	filter.Limit = limit

	users, total, err := uc.repo.FindAll(ctx, filter)
	if err != nil {
		slog.Error("failed to list users", "error", err)
		return nil, apperror.NewInternal()
	}

	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = userToDTO(u)
	}
	return &ListUsersOutput{
		Users:      dtos,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
	}, nil
}
