package judge

import (
	"context"

	adaptersubmission "github.com/training-judge-center/backend/internal/adapter/submission"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

type SubmissionUpdater struct {
	repo *adaptersubmission.Repository
}

func NewSubmissionUpdater(repo *adaptersubmission.Repository) *SubmissionUpdater {
	return &SubmissionUpdater{repo: repo}
}

func (u *SubmissionUpdater) GetByID(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *SubmissionUpdater) Update(ctx context.Context, s *submission.Submission) error {
	return u.repo.Update(ctx, s)
}
