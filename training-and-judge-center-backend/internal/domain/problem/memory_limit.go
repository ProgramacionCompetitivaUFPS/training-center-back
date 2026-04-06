package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type MemoryLimit struct {
	value int
}

func NewMemoryLimit(value int, maxGlobal int) (MemoryLimit, error) {
	if value <= 0 {
		return MemoryLimit{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "memoryLimit", Message: "Memory limit must be positive"},
		})
	}

	if value > maxGlobal {
		return MemoryLimit{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "memoryLimit", Message: "Exceeds global maximum memory limit"},
		})
	}

	return MemoryLimit{value: value}, nil
}

func (m MemoryLimit) Megabytes() int {
	return m.value
}

func RestoreMemoryLimit(value int) MemoryLimit {
	return MemoryLimit{value: value}
}
