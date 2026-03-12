package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type TimeLimit struct {
	value int
}

func NewTimeLimit(value int, maxGlobal int) (TimeLimit, error) {
	if value <= 0 {
		return TimeLimit{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "timeLimit", Message: "Time limit must be positive"},
		})
	}

	if value > maxGlobal {
		return TimeLimit{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "timeLimit", Message: "Exceeds global maximum time limit"},
		})
	}

	return TimeLimit{value: value}, nil
}

func (t TimeLimit) Value() int {
	return t.value
}
