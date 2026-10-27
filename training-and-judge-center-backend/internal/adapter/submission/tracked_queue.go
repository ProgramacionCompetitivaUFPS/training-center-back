package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

var _ appSubmission.SubmissionQueue = (*TrackedQueue)(nil)

// TrackedQueue records in the database which submissions the broker has
// confirmed, so a PENDING submission whose message never arrived can be told
// apart from one that is merely waiting its turn.
type TrackedQueue struct {
	db    infraPostgres.Querier
	inner appSubmission.SubmissionQueue
}

func NewTrackedQueue(db infraPostgres.Querier, inner appSubmission.SubmissionQueue) *TrackedQueue {
	return &TrackedQueue{db: db, inner: inner}
}

func (t *TrackedQueue) Publish(ctx context.Context, msg appSubmission.SubmissionQueueMessage) error {
	if err := t.inner.Publish(ctx, msg); err != nil {
		return err
	}
	q := infraPostgres.GetQuerier(ctx, t.db)
	if _, err := q.Exec(ctx, `UPDATE submissions SET queued_at = now() WHERE id = $1`, msg.SubmissionID); err != nil {
		// The message is in the queue; the worst case is one duplicate when the
		// recoverer later sees queued_at unset, which the worker discards.
		slog.ErrorContext(ctx, "queue: failed to record that the submission was queued", "submission_id", msg.SubmissionID, "error", err)
	}
	return nil
}

// Unqueue marks a still-PENDING submission as not queued, for the case where its
// message was consumed but judging failed before the submission started.
func (t *TrackedQueue) Unqueue(ctx context.Context, submissionID string) {
	q := infraPostgres.GetQuerier(ctx, t.db)
	if _, err := q.Exec(ctx, `
		UPDATE submissions SET queued_at = NULL, updated_at = now()
		WHERE id = $1 AND status = 'PENDING'
	`, submissionID); err != nil {
		slog.ErrorContext(ctx, "queue: failed to mark the submission as not queued", "submission_id", submissionID, "error", err)
	}
}
