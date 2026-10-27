package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type SubmissionUpdater interface {
	GetByID(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error)
	Update(ctx context.Context, s *submission.Submission) error
	// Claim moves a submission from PENDING to RUNNING in one atomic statement and
	// reports whether this call was the one that did it.
	Claim(ctx context.Context, id submission.SubmissionID) (bool, error)
}
