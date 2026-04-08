package user

import (
	"fmt"
	"strings"
)

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
	return "", fmt.Errorf("invalid sort field: %q", s)
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
	return "", fmt.Errorf("invalid sort order: %q", s)
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
	return "", fmt.Errorf("invalid search field: %q", s)
}

type UserFilter struct {
	Roles       []Role
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

func NewUserFilter(roles []Role, status *Status, country, city, institution string, searchField SearchField, searchTerm string, sort SortField, order SortOrder, page, limit int) (UserFilter, error) {
	if page < 1 {
		return UserFilter{}, fmt.Errorf("page must be >= 1, got %d", page)
	}
	if limit < 1 || limit > 100 {
		return UserFilter{}, fmt.Errorf("limit must be between 1 and 100, got %d", limit)
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
