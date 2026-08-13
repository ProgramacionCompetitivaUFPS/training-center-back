package problem

import "context"

type ProblemValidationRepository interface {
	Save(ctx context.Context, v *ProblemValidation) error
	FindByID(ctx context.Context, id string) (*ProblemValidation, error)
	FindLatestByProblemID(ctx context.Context, problemID string) (*ProblemValidation, bool, error)
}
