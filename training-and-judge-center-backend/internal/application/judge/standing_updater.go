package judge

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

type RecordVerdictRequest struct {
	UserID      shared.UserID
	ContestID   submission.ContestID
	ProblemID   submission.ProblemID
	Verdict     submission.SubmissionStatus
	SubmittedAt time.Time
}

type StandingUpdater interface {
	RecordVerdict(ctx context.Context, req RecordVerdictRequest) error
}
