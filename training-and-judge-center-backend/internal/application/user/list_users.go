package user

import (
	"context"
	"log/slog"
	"strings"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	maxPageLimit = 100
	DefaultPage  = 1
	DefaultLimit = 20
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
	repo user.Repository
}

func NewListUsersUseCase(repo user.Repository) *ListUsersUseCase {
	return &ListUsersUseCase{repo: repo}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, input ListUsersInput) (*ListUsersOutput, error) {
	var roles []shared.Role
	var status *user.Status
	var fieldErrors []apperror.FieldError

	for _, r := range input.Roles {
		role, err := shared.NewRole(strings.TrimSpace(r))
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{
				Field:   "role",
				Message: "Role must be one of: ADMIN, COACH, CONTESTANT",
			})
			break
		}
		roles = append(roles, role)
	}

	if input.Status != "" {
		s, err := user.NewStatus(input.Status)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{
				Field:   "status",
				Message: "Status must be ACTIVE or DEACTIVATED",
			})
		} else {
			status = &s
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

	if err := appshared.ValidatePagination(input.Page, input.Limit, maxPageLimit); err != nil {
		return nil, err
	}
	page := input.Page
	limit := input.Limit

	builtFilter, err := user.NewUserFilter(
		roles, status,
		input.Country, input.City, input.Institution,
		searchField, input.SearchTerm,
		sortField, sortOrder,
		page, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build user filter", "error", err)
		return nil, apperror.NewInternal()
	}

	users, total, err := uc.repo.FindAll(ctx, builtFilter)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list users", "error", err)
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
