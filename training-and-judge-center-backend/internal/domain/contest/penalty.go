package contest

import (
	"fmt"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	DefaultPenalty = 20
	MinPenalty     = 0
	MaxPenalty     = 1440
)

type Penalty struct {
	value int
}

func NewPenalty(v int) (Penalty, error) {
	if v < MinPenalty || v > MaxPenalty {
		return Penalty{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "penalty", Message: fmt.Sprintf("penalty must be between %d and %d minutes", MinPenalty, MaxPenalty)},
		})
	}
	return Penalty{value: v}, nil
}

func RestorePenalty(v int) Penalty {
	return Penalty{value: v}
}

func (p Penalty) Value() int { return p.value }
