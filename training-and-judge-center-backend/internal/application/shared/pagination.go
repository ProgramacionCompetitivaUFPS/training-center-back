package shared

import (
	"fmt"
	"math"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func ValidatePagination(page, limit, maxLimit int) error {
	if page < 1 {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "page must be a positive integer"},
		})
	}
	if limit < 1 || limit > maxLimit {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and %d", maxLimit)},
		})
	}
	return nil
}

func CalcTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}
