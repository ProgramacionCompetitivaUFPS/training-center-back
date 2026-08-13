package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type Solution struct {
	FileKey  string
	Language submission.Language
}

type SolutionProvider interface {
	GetSolutions(ctx context.Context, problemID string) ([]Solution, error)
}
