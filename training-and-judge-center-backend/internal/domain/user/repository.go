package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository interface {
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email Email) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByNickname(ctx context.Context, nickname Nickname) (*User, error)
	FindAll(ctx context.Context, filter UserFilter) ([]*User, int, error)
}

type SortField string

const (
	SortByCreatedAt     SortField = "createdAt"
	SortByName          SortField = "name"
	SortByNickname      SortField = "nickname"
	SortByEmail         SortField = "email"
	SortByDeactivatedAt SortField = "deactivatedAt"
)

func NewSortField(s string) (SortField, error) {
	switch SortField(s) {
	case SortByCreatedAt, SortByName, SortByNickname, SortByEmail, SortByDeactivatedAt:
		return SortField(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "sort", Message: "invalid sort field: " + s},
	})
}

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

func NewSortOrder(s string) (SortOrder, error) {
	switch SortOrder(strings.ToLower(s)) {
	case SortOrderAsc, SortOrderDesc:
		return SortOrder(strings.ToLower(s)), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "order", Message: "invalid sort order: " + s},
	})
}

type SearchField string

const (
	SearchByName        SearchField = "name"
	SearchByNickname    SearchField = "nickname"
	SearchByEmail       SearchField = "email"
	SearchByInstitution SearchField = "institution"
	SearchByAll         SearchField = "all"
)

func NewSearchField(s string) (SearchField, error) {
	switch SearchField(s) {
	case SearchByName, SearchByNickname, SearchByEmail, SearchByInstitution, SearchByAll:
		return SearchField(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "searchField", Message: "invalid search field: " + s},
	})
}

type UserFilter struct {
	Roles       []shared.Role
	Status      *Status
	Country     string
	City        string
	Institution string
	SearchField SearchField
	SearchTerm  string
	Sort        SortField
	Order       SortOrder
	Page        int
	Limit       int
}

func NewUserFilter(roles []shared.Role, status *Status, country, city, institution string, searchField SearchField, searchTerm string, sort SortField, order SortOrder, page, limit int) (UserFilter, error) {
	if page < 1 {
		return UserFilter{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: fmt.Sprintf("page must be >= 1, got %d", page)},
		})
	}
	if limit < 1 || limit > 100 {
		return UserFilter{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and 100, got %d", limit)},
		})
	}
	return UserFilter{
		Roles:       roles,
		Status:      status,
		Country:     country,
		City:        city,
		Institution: institution,
		SearchField: searchField,
		SearchTerm:  searchTerm,
		Sort:        sort,
		Order:       order,
		Page:        page,
		Limit:       limit,
	}, nil
}
