package group

import (
	"fmt"
	"math"
	"strings"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validatePagination(page, limit int) error {
	if page < 1 {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "page must be a positive integer"},
		})
	}
	if limit < 1 || limit > MaxPageLimit {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and %d", MaxPageLimit)},
		})
	}
	return nil
}

func calcTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}

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
