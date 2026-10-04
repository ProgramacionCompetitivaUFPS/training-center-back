package submission

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appJudge.StalePendingRecoverer = (*StalePendingRecoverer)(nil)

const (
	// maxRequeues is how many successful re-publishes one submission gets
	// before it is failed; each one was a message that went missing again.
	maxRequeues = 5
	// maxRequeuesPerSweep keeps one sweep from flooding the queue.
	maxRequeuesPerSweep = 100
)

// StalePendingRecoverer re-publishes PENDING submissions whose message the
// broker never confirmed (queued_at is NULL). A submission that is merely
// waiting in a long queue has queued_at set and is never touched, however long
// the wait: the queue is durable, so a confirmed message is not lost.
type StalePendingRecoverer struct {
	db    infraPostgres.Querier
	queue appSubmission.SubmissionQueue // a TrackedQueue, which sets queued_at on success
}

func NewStalePendingRecoverer(db infraPostgres.Querier, queue appSubmission.SubmissionQueue) *StalePendingRecoverer {
	return &StalePendingRecoverer{db: db, queue: queue}
}

// RecoverStalePending first fails the unqueued submissions already re-published
// maxRequeues times, then re-publishes a bounded batch of the rest. cutoff is the
// grace period that lets an API request finish its own publish. The sweep stops
// at the first publish failure: the broker is down and the next sweep retries.
func (r *StalePendingRecoverer) RecoverStalePending(ctx context.Context, cutoff time.Time) (int, int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	tag, err := q.Exec(ctx, `
		UPDATE submissions
		SET status = 'SYSTEM_ERROR', updated_at = now()
		WHERE status = 'PENDING' AND queued_at IS NULL AND updated_at < $1 AND requeue_count >= $2
	`, cutoff, maxRequeues)
	if err != nil {
		slog.ErrorContext(ctx, "recoverer: failed to fail exhausted pending submissions", "error", err)
		return 0, 0, apperror.NewInternal()
	}
	failed := int(tag.RowsAffected())

	rows, err := q.Query(ctx, `
		SELECT id, user_id, contest_id, COALESCE(problem_id::text, ''), language
		FROM submissions
		WHERE status = 'PENDING' AND queued_at IS NULL AND updated_at < $1 AND requeue_count < $2
		ORDER BY updated_at
		LIMIT $3
	`, cutoff, maxRequeues, maxRequeuesPerSweep)
	if err != nil {
		slog.ErrorContext(ctx, "recoverer: failed to list unqueued submissions", "error", err)
		return 0, failed, apperror.NewInternal()
	}
	defer rows.Close()

	var found []appSubmission.SubmissionQueueMessage
	for rows.Next() {
		var id, userID, problemID, language string
		var contestID *string
		if err := rows.Scan(&id, &userID, &contestID, &problemID, &language); err != nil {
			slog.ErrorContext(ctx, "recoverer: failed to scan unqueued submission", "error", err)
			return 0, failed, apperror.NewInternal()
		}
		priority := appSubmission.QueuePriorityPractice
		if contestID != nil {
			priority = appSubmission.QueuePriorityContest
		}
		found = append(found, appSubmission.SubmissionQueueMessage{
			SubmissionID: id,
			Priority:     priority,
			EnqueuedAt:   time.Now(),
			Metadata: appSubmission.SubmissionQueueMetadata{
				ContestID: contestID, ProblemID: problemID, UserID: userID, Language: language,
			},
		})
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "recoverer: error iterating unqueued submissions", "error", err)
		return 0, failed, apperror.NewInternal()
	}
	rows.Close()

	requeued := 0
	for _, msg := range found {
		if err := r.queue.Publish(ctx, msg); err != nil {
			slog.ErrorContext(ctx, "recoverer: failed to re-enqueue submission, stopping this sweep", "submission_id", msg.SubmissionID, "error", err)
			break
		}
		if _, err := q.Exec(ctx, `UPDATE submissions SET requeue_count = requeue_count + 1 WHERE id = $1`, msg.SubmissionID); err != nil {
			slog.ErrorContext(ctx, "recoverer: failed to count the re-enqueue", "submission_id", msg.SubmissionID, "error", err)
		}
		requeued++
	}
	return requeued, failed, nil
}
