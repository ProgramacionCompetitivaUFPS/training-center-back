package material

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

var tagRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*[a-z0-9]$`)

// maxTags bounds how many distinct tags a single material can carry. Tags are meant to
// categorize, not to duplicate full-text search (title/content already cover that) — past
// this many, they stop narrowing anything down and just add noise to DistinctTags.
const maxTags = 20

type Tags struct {
	values []string
}

func NewTags(values []string) (Tags, error) {
	if len(values) == 0 {
		return Tags{values: []string{}}, nil
	}

	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	var invalid []string

	for _, tag := range values {
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}

		runes := []rune(tag)
		if len(runes) < 2 || len(runes) > 50 {
			invalid = append(invalid, tag)
			continue
		}
		if strings.Contains(tag, "--") || strings.Contains(tag, "__") {
			invalid = append(invalid, tag)
			continue
		}
		if !tagRegex.MatchString(tag) {
			invalid = append(invalid, tag)
			continue
		}

		deduped = append(deduped, tag)
	}

	if len(invalid) > 0 {
		var fieldErrs []apperror.FieldError
		for _, t := range invalid {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "tags",
				Message: "invalid tag: " + t,
			})
		}
		return Tags{}, apperror.NewValidation(fieldErrs)
	}

	if len(deduped) > maxTags {
		return Tags{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "tags", Message: fmt.Sprintf("too many tags: max %d, got %d", maxTags, len(deduped))},
		})
	}

	return Tags{values: deduped}, nil
}

func RestoreTags(values []string) Tags {
	if values == nil {
		return Tags{values: []string{}}
	}
	cp := make([]string, len(values))
	copy(cp, values)
	return Tags{values: cp}
}

func (t Tags) Values() []string {
	cp := make([]string, len(t.values))
	copy(cp, t.values)
	return cp
}
