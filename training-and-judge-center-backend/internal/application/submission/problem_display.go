package submission

import (
	"context"
	"errors"

	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// resolveProblemDisplay fetches the live problem display for a submission.
// When the problem has been deleted (problemID is empty or the provider returns
// not-found), it falls back to the snapshot stored on the submission itself.
func resolveProblemDisplay(ctx context.Context, provider ProblemDisplayProvider, sub *domainSubmission.Submission) (*ProblemDisplay, error) {
	if pid := sub.ProblemID(); pid != "" {
		p, err := provider.GetProblemByID(ctx, pid)
		if err != nil {
			var ae *apperror.AppError
			if errors.As(err, &ae) && ae.Kind == apperror.KindNotFound {
				return &ProblemDisplay{Slug: sub.ProblemSlug(), Title: sub.ProblemTitle()}, nil
			}
			return nil, err
		}
		return p, nil
	}
	return &ProblemDisplay{Slug: sub.ProblemSlug(), Title: sub.ProblemTitle()}, nil
}
