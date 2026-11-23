//go:build integration

package submission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

type itQueue struct {
	pool     *pgxpool.Pool
	failFor  map[string]bool
	seen     []string
	statuses map[string]string
}

func newItQueue(pool *pgxpool.Pool) *itQueue {
	return &itQueue{pool: pool, failFor: map[string]bool{}, statuses: map[string]string{}}
}

func (q *itQueue) Publish(ctx context.Context, msg appSubmission.SubmissionQueueMessage) error {
	if q.failFor[msg.SubmissionID] {
		return errors.New("broker down")
	}
	var status string
	if q.pool != nil {
		_ = q.pool.QueryRow(ctx, `SELECT status FROM submissions WHERE id = $1`, msg.SubmissionID).Scan(&status)
	}
	q.statuses[msg.SubmissionID] = status
	q.seen = append(q.seen, msg.SubmissionID)
	return nil
}

func itQueuedAt(t *testing.T, pool *pgxpool.Pool, id string) *time.Time {
	t.Helper()
	var at *time.Time
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT queued_at FROM submissions WHERE id = $1`, id).Scan(&at))
	return at
}

func itSetState(t *testing.T, pool *pgxpool.Pool, id string, sql string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `UPDATE submissions SET `+sql+` WHERE id = '`+id+`'`, args...)
	require.NoError(t, err)
}

func TestTrackedQueue_Publish_RecordsQueuedAtOnlyWhenTheBrokerConfirms(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "ok", ProblemID: itProblem1, UserID: itUserA})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "lost", ProblemID: itProblem1, UserID: itUserA})
	inner := newItQueue(nil)
	inner.failFor["lost"] = true
	tracked := NewTrackedQueue(pool, inner)

	require.NoError(t, tracked.Publish(context.Background(), appSubmission.SubmissionQueueMessage{SubmissionID: "ok"}))
	err := tracked.Publish(context.Background(), appSubmission.SubmissionQueueMessage{SubmissionID: "lost"})

	require.Error(t, err)
	assert.NotNil(t, itQueuedAt(t, pool, "ok"))
	assert.Nil(t, itQueuedAt(t, pool, "lost"))
}

func TestTrackedQueue_Unqueue_ClearsQueuedAtOnlyForPending(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "pending", ProblemID: itProblem1, UserID: itUserA})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "running", ProblemID: itProblem1, UserID: itUserA, Status: "RUNNING"})
	itSetState(t, pool, "pending", "queued_at = now()")
	itSetState(t, pool, "running", "queued_at = now()")
	tracked := NewTrackedQueue(pool, newItQueue(nil))

	tracked.Unqueue(context.Background(), "pending")
	tracked.Unqueue(context.Background(), "running")

	assert.Nil(t, itQueuedAt(t, pool, "pending"))
	assert.NotNil(t, itQueuedAt(t, pool, "running"))
}

func TestRejudger_ListByProblemBefore_SkipsPendingRunningAndNewer(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	at := itNow
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "done", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED", SubmittedAt: at})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "wa", ProblemID: itProblem1, UserID: itUserB, Status: "WRONG_ANSWER", SubmittedAt: at, Language: "java17"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "pending", ProblemID: itProblem1, UserID: itUserA, SubmittedAt: at})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "running", ProblemID: itProblem1, UserID: itUserA, Status: "RUNNING", SubmittedAt: at})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "newer", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED", SubmittedAt: at.Add(2 * time.Hour)})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "other-problem", ProblemID: itProblem2, UserID: itUserA, Status: "ACCEPTED", SubmittedAt: at})
	rejudger := NewRejudger(pool, newItQueue(pool))

	got, err := rejudger.ListByProblemBefore(context.Background(), itProblem1, at.Add(time.Hour))

	require.NoError(t, err)
	ids := make([]string, len(got))
	for i, g := range got {
		ids[i] = g.ID
	}
	assert.ElementsMatch(t, []string{"done", "wa"}, ids)
}

func TestRejudger_RejudgeBatch_ResetsBeforePublishingAndSkipsInProgress(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "done", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "running", ProblemID: itProblem1, UserID: itUserA, Status: "RUNNING"})
	itSetState(t, pool, "done", "judged_at = now(), time_ms = 90, memory_kb = 512, compile_log = 'log', queued_at = now(), requeue_count = 3")
	queue := newItQueue(pool)
	rejudger := NewRejudger(pool, queue)
	subs := []appProblem.SubmissionRejudgeInfo{
		{ID: "done", UserID: itUserA, Language: "cpp20"},
		{ID: "running", UserID: itUserA, Language: "cpp20"},
	}

	queued, err := rejudger.RejudgeBatch(context.Background(), subs, itProblem1, itNow)

	require.NoError(t, err)
	assert.Equal(t, 1, queued)
	assert.Equal(t, []string{"done"}, queue.seen)
	assert.Equal(t, "PENDING", queue.statuses["done"], "the reset must be visible before the message is published")
	assert.Equal(t, "RUNNING", itStatusOf(t, pool, "running"))
	var judgedAt *time.Time
	var timeMs, memoryKb *int
	var log *string
	var requeues int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT judged_at, time_ms, memory_kb, compile_log, requeue_count FROM submissions WHERE id = 'done'`,
	).Scan(&judgedAt, &timeMs, &memoryKb, &log, &requeues))
	assert.Nil(t, judgedAt)
	assert.Nil(t, timeMs)
	assert.Nil(t, memoryKb)
	assert.Nil(t, log)
	assert.Equal(t, 0, requeues)
	assert.Nil(t, itQueuedAt(t, pool, "done"))
}

func TestRejudger_RejudgeBatch_RestoresTheSubmissionWhosePublishFails(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "good", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "bad", ProblemID: itProblem1, UserID: itUserA, Status: "WRONG_ANSWER"})
	itSetState(t, pool, "bad", "time_ms = 77, memory_kb = 100, compile_log = 'prev'")
	queue := newItQueue(pool)
	queue.failFor["bad"] = true
	rejudger := NewRejudger(pool, queue)

	queued, err := rejudger.RejudgeBatch(context.Background(), []appProblem.SubmissionRejudgeInfo{
		{ID: "good", UserID: itUserA, Language: "cpp20"},
		{ID: "bad", UserID: itUserA, Language: "cpp20"},
	}, itProblem1, itNow)

	require.NoError(t, err)
	assert.Equal(t, 1, queued)
	assert.Equal(t, "PENDING", itStatusOf(t, pool, "good"))
	assert.Equal(t, "WRONG_ANSWER", itStatusOf(t, pool, "bad"))
	var timeMs int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT time_ms FROM submissions WHERE id = 'bad'`).Scan(&timeMs))
	assert.Equal(t, 77, timeMs)
}

func TestRejudger_RejudgeByID_InProgressSubmissionIsSkippedWithoutError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "running", ProblemID: itProblem1, UserID: itUserA, Status: "RUNNING"})
	queue := newItQueue(pool)

	err := NewRejudger(pool, queue).RejudgeByID(context.Background(), "running", itProblem1, itUserA, nil, "cpp20", itNow)

	require.NoError(t, err)
	assert.Empty(t, queue.seen)
	assert.Equal(t, "RUNNING", itStatusOf(t, pool, "running"))
}

func TestRejudger_RejudgeByID_PublishFailureRestoresAndReturnsInternal(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "done", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED"})
	queue := newItQueue(pool)
	queue.failFor["done"] = true

	err := NewRejudger(pool, queue).RejudgeByID(context.Background(), "done", itProblem1, itUserA, nil, "cpp20", itNow)

	require.Error(t, err)
	assert.Equal(t, "ACCEPTED", itStatusOf(t, pool, "done"))
}

func TestStalePendingRecoverer_RequeuesOnlyUnconfirmedAndOldPending(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	for _, id := range []string{"lost", "waiting", "fresh"} {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: itProblem1, UserID: itUserA})
	}
	cutoff := itNow
	itSetState(t, pool, "lost", "queued_at = NULL, updated_at = $1", cutoff.Add(-time.Hour))
	itSetState(t, pool, "waiting", "queued_at = now(), updated_at = $1", cutoff.Add(-time.Hour))
	itSetState(t, pool, "fresh", "queued_at = NULL, updated_at = $1", cutoff.Add(time.Hour))
	queue := newItQueue(nil)

	requeued, failed, err := NewStalePendingRecoverer(pool, queue).RecoverStalePending(context.Background(), cutoff)

	require.NoError(t, err)
	assert.Equal(t, 1, requeued)
	assert.Equal(t, 0, failed)
	assert.Equal(t, []string{"lost"}, queue.seen)
	var count int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT requeue_count FROM submissions WHERE id = 'lost'`).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestStalePendingRecoverer_FailsTheSubmissionThatExhaustedItsRequeues(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "exhausted", ProblemID: itProblem1, UserID: itUserA})
	itSetState(t, pool, "exhausted", "queued_at = NULL, requeue_count = $1, updated_at = $2", maxRequeues, itNow.Add(-time.Hour))
	queue := newItQueue(nil)

	requeued, failed, err := NewStalePendingRecoverer(pool, queue).RecoverStalePending(context.Background(), itNow)

	require.NoError(t, err)
	assert.Equal(t, 0, requeued)
	assert.Equal(t, 1, failed)
	assert.Empty(t, queue.seen)
	assert.Equal(t, "SYSTEM_ERROR", itStatusOf(t, pool, "exhausted"))
}

func TestStalePendingRecoverer_StopsTheSweepAtTheFirstPublishFailure(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	for _, id := range []string{"first", "second", "third"} {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: itProblem1, UserID: itUserA})
	}
	itSetState(t, pool, "first", "updated_at = $1", itNow.Add(-3*time.Hour))
	itSetState(t, pool, "second", "updated_at = $1", itNow.Add(-2*time.Hour))
	itSetState(t, pool, "third", "updated_at = $1", itNow.Add(-time.Hour))
	queue := newItQueue(nil)
	queue.failFor["second"] = true

	requeued, _, err := NewStalePendingRecoverer(pool, queue).RecoverStalePending(context.Background(), itNow)

	require.NoError(t, err)
	assert.Equal(t, 1, requeued)
	assert.Equal(t, []string{"first"}, queue.seen)
}
