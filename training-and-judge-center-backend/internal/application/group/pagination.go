package group

import (
	"fmt"
	"strings"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	maxPageLimit         = 50
	DefaultPageLimit     = 20
	maxRequestsPageLimit = 100
)

func parseSort(raw string, def domainGroup.SortField, allowed []domainGroup.SortField) (domainGroup.SortField, error) {
	if raw == "" {
		return def, nil
	}
	for _, f := range allowed {
		if string(f) == raw {
			return f, nil
		}
	}
	names := make([]string, len(allowed))
	for i, f := range allowed {
		names[i] = string(f)
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "sortBy", Message: fmt.Sprintf("invalid sortBy; must be one of: %s", strings.Join(names, ", "))},
	})
}

func parseOrder(raw string) (domainGroup.SortOrder, error) {
	switch raw {
	case "":
		return domainGroup.OrderAsc, nil
	case "asc":
		return domainGroup.OrderAsc, nil
	case "desc":
		return domainGroup.OrderDesc, nil
	default:
		return "", apperror.NewValidation([]apperror.FieldError{
			{Field: "order", Message: "invalid order; must be asc or desc"},
		})
	}
}
