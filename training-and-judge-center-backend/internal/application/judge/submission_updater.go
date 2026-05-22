package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type SubmissionUpdater interface {
	GetByID(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error)
	Update(ctx context.Context, s *submission.Submission) error
}
