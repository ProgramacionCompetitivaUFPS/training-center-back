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
