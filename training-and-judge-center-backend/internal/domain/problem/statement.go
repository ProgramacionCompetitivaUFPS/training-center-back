package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type Statement struct {
	value *string
}

func NewStatement(value *string) (Statement, error) {
	if value != nil && len([]rune(*value)) > 150_000 {
		return Statement{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "statement", Message: "Statement must not exceed 150,000 characters"},
		})
	}
	return Statement{value: value}, nil
}

func RestoreStatement(value *string) Statement {
	return Statement{value: value}
}

func (s Statement) Value() *string {
	return s.value
}
