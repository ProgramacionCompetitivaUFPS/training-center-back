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
	// maxRequeues is how many times one submission is re-published before it
	// is failed; together with the sweep interval it bounds the retries.
	maxRequeues = 3
	// maxRequeuesPerSweep keeps one sweep from flooding the queue.
	maxRequeuesPerSweep = 100
)

type StalePendingRecoverer struct {
	db    infraPostgres.Querier
	queue appSubmission.SubmissionQueue
}

func NewStalePendingRecoverer(db infraPostgres.Querier, queue appSubmission.SubmissionQueue) *StalePendingRecoverer {
	return &StalePendingRecoverer{db: db, queue: queue}
}

// RecoverStalePending first fails the PENDING submissions already re-published
// maxRequeues times, then claims a bounded batch of the rest (bumping their
// counter and updated_at in the same statement, so each is retried at most once
// per staleness window) and publishes it. The claim commits before the publish,
// the same order the rejudger needs: the worker drops a message for a
// submission that is not PENDING.
func (r *StalePendingRecoverer) RecoverStalePending(ctx context.Context, cutoff time.Time) (int, int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	tag, err := q.Exec(ctx, `
		UPDATE submissions
		SET status = 'SYSTEM_ERROR', updated_at = now()
		WHERE status = 'PENDING' AND updated_at < $1 AND requeue_count >= $2
	`, cutoff, maxRequeues)
	if err != nil {
		slog.ErrorContext(ctx, "recoverer: failed to fail exhausted pending submissions", "error", err)
		return 0, 0, apperror.NewInternal()
	}
	failed := int(tag.RowsAffected())

	rows, err := q.Query(ctx, `
		UPDATE submissions
		SET requeue_count = requeue_count + 1, updated_at = now()
		WHERE id IN (
			SELECT id FROM submissions
			WHERE status = 'PENDING' AND updated_at < $1 AND requeue_count < $2
			ORDER BY updated_at
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, user_id, contest_id, COALESCE(problem_id::text, ''), language
	`, cutoff, maxRequeues, maxRequeuesPerSweep)
	if err != nil {
		slog.ErrorContext(ctx, "recoverer: failed to claim pending submissions", "error", err)
		return 0, failed, apperror.NewInternal()
	}
	defer rows.Close()

	var claimed []appSubmission.SubmissionQueueMessage
	for rows.Next() {
		var id, userID, problemID, language string
		var contestID *string
		if err := rows.Scan(&id, &userID, &contestID, &problemID, &language); err != nil {
			slog.ErrorContext(ctx, "recoverer: failed to scan pending submission", "error", err)
			return 0, failed, apperror.NewInternal()
		}
		priority := appSubmission.QueuePriorityPractice
		if contestID != nil {
			priority = appSubmission.QueuePriorityContest
		}
		claimed = append(claimed, appSubmission.SubmissionQueueMessage{
			SubmissionID: id,
			Priority:     priority,
			EnqueuedAt:   time.Now(),
			Metadata: appSubmission.SubmissionQueueMetadata{
				ContestID: contestID, ProblemID: problemID, UserID: userID, Language: language,
			},
		})
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "recoverer: error iterating pending submissions", "error", err)
		return 0, failed, apperror.NewInternal()
	}
	rows.Close()

	requeued := 0
	for _, msg := range claimed {
		if err := r.queue.Publish(ctx, msg); err != nil {
			// Already counted: the next window retries it, up to maxRequeues.
			slog.ErrorContext(ctx, "recoverer: failed to re-enqueue submission", "submission_id", msg.SubmissionID, "error", err)
			continue
		}
		requeued++
	}
	return requeued, failed, nil
}
