package submission

import (
	"context"

	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
)

// resolveProblemDisplay fetches the live problem display for a submission.
// Falls back to the snapshot stored on the submission when the problem has been
// deleted (problem_id NULL → empty string) or the provider is unavailable.
func resolveProblemDisplay(ctx context.Context, provider ProblemDisplayProvider, sub *domainSubmission.Submission) (*ProblemDisplay, error) {
	if pid := sub.ProblemID(); pid != "" {
		if p, err := provider.GetProblemByID(ctx, pid); err == nil {
			return p, nil
		}
	}
	return &ProblemDisplay{Slug: sub.ProblemSlug(), Title: sub.ProblemTitle()}, nil
}
