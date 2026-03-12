package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type Tags struct {
	values []string
}

func NewTags(values []string, allowedTags map[string]struct{}) (Tags, error) {
	if values == nil {
		return Tags{values: []string{}}, nil
	}

	seen := make(map[string]struct{}, len(values))
	var fieldErrs []apperror.FieldError

	for _, tag := range values {
		if tag == "" {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "tags",
				Message: "Tag cannot be empty",
			})
			continue
		}

		if _, exists := seen[tag]; exists {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "tags",
				Message: "Duplicate tag: " + tag,
			})
			continue
		}

		if _, valid := allowedTags[tag]; !valid {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "tags",
				Message: "Invalid tag: " + tag,
			})
			continue
		}

		seen[tag] = struct{}{}
	}

	if len(fieldErrs) > 0 {
		return Tags{}, apperror.NewValidation(fieldErrs)
	}

	return Tags{values: values}, nil
}

func (t Tags) Values() []string {
	return t.values
}
