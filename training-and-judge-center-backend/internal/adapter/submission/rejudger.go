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

// previousState is what a submission looked like before its reset to PENDING,
// kept so a failed publish can put it back.
type previousState struct {
	id         string
	status     string
	judgedAt   *time.Time
	timeMs     *int
	memoryKb   *int
	compileLog *string
}

// resetToPending moves the given finished submissions back to PENDING and
// returns their previous state. It must commit before anything is published:
// the worker discards a message whose submission is not PENDING yet.
func (r *Rejudger) resetToPending(ctx context.Context, ids []string) ([]previousState, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx, `
		WITH prev AS (
			SELECT id, status, judged_at, time_ms, memory_kb, compile_log
			FROM submissions
			WHERE id = ANY($1::text[]) AND status NOT IN ('PENDING', 'RUNNING')
			FOR UPDATE
		)
		UPDATE submissions s
		SET status = 'PENDING', judged_at = NULL, time_ms = NULL, memory_kb = NULL, compile_log = NULL
		FROM prev
		WHERE s.id = prev.id
		RETURNING prev.id, prev.status, prev.judged_at, prev.time_ms, prev.memory_kb, prev.compile_log
	`, ids)
	if err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to reset submissions", "count", len(ids), "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []previousState
	for rows.Next() {
		var p previousState
		if err := rows.Scan(&p.id, &p.status, &p.judgedAt, &p.timeMs, &p.memoryKb, &p.compileLog); err != nil {
			slog.ErrorContext(ctx, "rejudger: failed to scan reset row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "rejudger: error iterating reset rows", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

// restore undoes resetToPending for a submission whose message never reached
// the queue, so it is not left PENDING with nothing to judge it.
func (r *Rejudger) restore(ctx context.Context, p previousState) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, `
		UPDATE submissions
		SET status = $2, judged_at = $3, time_ms = $4, memory_kb = $5, compile_log = $6
		WHERE id = $1 AND status = 'PENDING'
	`, p.id, p.status, p.judgedAt, p.timeMs, p.memoryKb, p.compileLog)
	if err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to restore submission after a failed enqueue", "submission_id", p.id, "error", err)
	}
}

func rejudgeMessage(sub appProblem.SubmissionRejudgeInfo, problemID string, now time.Time) appSubmission.SubmissionQueueMessage {
	return appSubmission.SubmissionQueueMessage{
		SubmissionID: sub.ID,
		Priority:     appSubmission.QueuePriorityRejudge,
		EnqueuedAt:   now,
		Metadata: appSubmission.SubmissionQueueMetadata{
			ContestID: sub.ContestID,
			ProblemID: problemID,
			UserID:    sub.UserID,
			Language:  sub.Language,
		},
	}
}

// RejudgeBatch resets the submissions to PENDING and then publishes them; the
// ones whose publish fails are put back as they were. It returns how many were queued.
func (r *Rejudger) RejudgeBatch(ctx context.Context, subs []appProblem.SubmissionRejudgeInfo, problemID string, now time.Time) (int, error) {
	ids := make([]string, len(subs))
	for i, sub := range subs {
		ids[i] = sub.ID
	}
	prev, err := r.resetToPending(ctx, ids)
	if err != nil {
		return 0, err
	}
	reset := make(map[string]previousState, len(prev))
	for _, p := range prev {
		reset[p.id] = p
	}

	queued := 0
	for _, sub := range subs {
		p, ok := reset[sub.ID]
		if !ok {
			continue
		}
		if err := r.queue.Publish(ctx, rejudgeMessage(sub, problemID, now)); err != nil {
			slog.ErrorContext(ctx, "rejudger: failed to enqueue submission", "submission_id", sub.ID, "error", err)
			r.restore(ctx, p)
			continue
		}
		queued++
	}
	return queued, nil
}

func (r *Rejudger) rejudgeOne(ctx context.Context, info appProblem.SubmissionRejudgeInfo, problemID string, now time.Time) error {
	prev, err := r.resetToPending(ctx, []string{info.ID})
	if err != nil {
		return err
	}
	if len(prev) == 0 {
		slog.WarnContext(ctx, "rejudger: submission already in progress, rejudge skipped", "submission_id", info.ID)
		return nil
	}

	if err := r.queue.Publish(ctx, rejudgeMessage(info, problemID, now)); err != nil {
		slog.ErrorContext(ctx, "rejudger: failed to enqueue submission", "submission_id", info.ID, "error", err)
		r.restore(ctx, prev[0])
		return apperror.NewInternal()
	}
	return nil
}
