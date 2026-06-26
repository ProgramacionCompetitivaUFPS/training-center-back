package submission

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Rejudger implements appProblem.SubmissionRejudger.
type Rejudger struct {
	db    infraPostgres.Querier
	queue appSubmission.SubmissionQueue
}

func NewRejudger(db infraPostgres.Querier, queue appSubmission.SubmissionQueue) *Rejudger {
	return &Rejudger{db: db, queue: queue}
}

func (r *Rejudger) ListByProblemBefore(ctx context.Context, problemID string, before time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx, `
		SELECT id, user_id, contest_id, language
		FROM submissions
		WHERE problem_id = $1 AND submitted_at < $2
		  AND status NOT IN ('PENDING', 'RUNNING')
	`, problemID, before)
	if err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to list submissions", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appProblem.SubmissionRejudgeInfo
	for rows.Next() {
		var info appProblem.SubmissionRejudgeInfo
		if err := rows.Scan(&info.ID, &info.UserID, &info.ContestID, &info.Language); err != nil {
			slog.ErrorContext(ctx, "rejudger: failed to scan row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, info)
	}
	if rows.Err() != nil {
		slog.ErrorContext(ctx, "rejudger: error iterating rows", "error", rows.Err())
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (r *Rejudger) ListByProblemAndContestBefore(ctx context.Context, problemID, contestID string, before time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx, `
		SELECT id, user_id, contest_id, language
		FROM submissions
		WHERE problem_id = $1 AND contest_id = $2 AND submitted_at < $3
		  AND status NOT IN ('PENDING', 'RUNNING')
	`, problemID, contestID, before)
	if err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to list contest submissions", "problem_id", problemID, "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appProblem.SubmissionRejudgeInfo
	for rows.Next() {
		var info appProblem.SubmissionRejudgeInfo
		if err := rows.Scan(&info.ID, &info.UserID, &info.ContestID, &info.Language); err != nil {
			slog.ErrorContext(ctx, "rejudger: failed to scan row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, info)
	}
	if rows.Err() != nil {
		slog.ErrorContext(ctx, "rejudger: error iterating rows", "error", rows.Err())
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (r *Rejudger) RejudgeByID(ctx context.Context, submissionID, problemID, userID string, contestID *string, language string, now time.Time) error {
	info := appProblem.SubmissionRejudgeInfo{
		ID:        submissionID,
		UserID:    userID,
		ContestID: contestID,
		Language:  language,
	}
	return r.rejudgeOne(ctx, info, problemID, now)
}

// RejudgeBatch publishes all submissions to the queue and resets their DB status in one batch UPDATE.
func (r *Rejudger) RejudgeBatch(ctx context.Context, subs []appProblem.SubmissionRejudgeInfo, problemID string, now time.Time) (int, error) {
	published := make([]string, 0, len(subs))
	for _, sub := range subs {
		if err := r.queue.Publish(ctx, appSubmission.SubmissionQueueMessage{
			SubmissionID: sub.ID,
			Priority:     appSubmission.QueuePriorityRejudge,
			EnqueuedAt:   now,
			Metadata: appSubmission.SubmissionQueueMetadata{
				ContestID: sub.ContestID,
				ProblemID: problemID,
				UserID:    sub.UserID,
				Language:  sub.Language,
			},
		}); err != nil {
			slog.ErrorContext(ctx, "rejudger: failed to enqueue submission", "submission_id", sub.ID, "error", err)
			continue
		}
		published = append(published, sub.ID)
	}

	if len(published) == 0 {
		return 0, nil
	}

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, `
		UPDATE submissions
		SET status = 'PENDING', judged_at = NULL, time_ms = NULL, memory_kb = NULL, compile_log = NULL
		WHERE id = ANY($1::uuid[]) AND status NOT IN ('PENDING', 'RUNNING')
	`, published)
	if err != nil {
		// Messages already in queue; judge handles them — log but don't fail.
		slog.ErrorContext(ctx, "rejudger: failed to batch reset submissions", "count", len(published), "error", err)
	}
	return len(published), nil
}

func (r *Rejudger) rejudgeOne(ctx context.Context, info appProblem.SubmissionRejudgeInfo, problemID string, now time.Time) error {
	if err := r.queue.Publish(ctx, appSubmission.SubmissionQueueMessage{
		SubmissionID: info.ID,
		Priority:     appSubmission.QueuePriorityRejudge,
		EnqueuedAt:   now,
		Metadata: appSubmission.SubmissionQueueMetadata{
			ContestID: info.ContestID,
			ProblemID: problemID,
			UserID:    info.UserID,
			Language:  info.Language,
		},
	}); err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to enqueue submission", "submission_id", info.ID, "error", err)
		return apperror.NewInternal()
	}

	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE submissions
		SET status = 'PENDING', judged_at = NULL, time_ms = NULL, memory_kb = NULL, compile_log = NULL
		WHERE id = $1 AND status NOT IN ('PENDING', 'RUNNING')
	`, info.ID)
	if err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to reset submission after enqueue", "submission_id", info.ID, "error", err)
		return nil // message already in queue; judge handles it
	}
	if tag.RowsAffected() == 0 {
		slog.WarnContext(ctx, "rejudger: submission already in progress, db reset skipped", "submission_id", info.ID)
	}
	return nil
}
