package submission

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type ListFilters struct {
	UserID    *shared.UserID
	ProblemID *ProblemID
	ContestID *ContestID
	Page      int
	Limit     int
}

type Repository interface {
	Save(ctx context.Context, s *Submission) error
	FindByID(ctx context.Context, id SubmissionID) (*Submission, error)
	List(ctx context.Context, filters ListFilters) ([]*Submission, int, error)
	FindLastByUserAndProblem(ctx context.Context, userID, problemID string) (*Submission, error)
	ExistsByHashAndUserAndProblem(ctx context.Context, fileHash, userID, problemID string) (bool, error)
	UpdateVisibility(ctx context.Context, s *Submission) error
}
