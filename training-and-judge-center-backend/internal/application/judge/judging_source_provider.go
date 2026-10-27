package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type JudgingSource struct {
	Filename string
	FileKey  string
	Language submission.Language
}

// JudgingSourceProvider reads a problem's checker/validator source, if
// uploaded. Both methods return (nil, nil) when nothing was uploaded for
// that role — it's optional, unlike a problem's solutions.
type JudgingSourceProvider interface {
	GetCheckerSource(ctx context.Context, problemID string) (*JudgingSource, error)
	GetValidatorSource(ctx context.Context, problemID string) (*JudgingSource, error)
}
