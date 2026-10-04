package submission

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func listedRows(ids ...string) pgx.Rows {
	rows := &mockRows{}
	for _, id := range ids {
		rows.scanFns = append(rows.scanFns, func(dest ...any) error {
			*(dest[0].(*string)) = id
			*(dest[1].(*string)) = testUserID
			*(dest[3].(*string)) = testProblemID
			*(dest[4].(*string)) = "cpp20"
			return nil
		})
	}
	return rows
}

func newRecoverer(q appSubmission.SubmissionQueue, listed ...string) (*StalePendingRecoverer, *[]string) {
	var execSQL []string
	r := NewStalePendingRecoverer(&mockQuerier{
		execFn: func(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			execSQL = append(execSQL, sql)
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
		queryFn: func(_ context.Context, sql string, _ ...interface{}) (pgx.Rows, error) {
			if !strings.Contains(sql, "queued_at IS NULL") {
				return nil, errors.New("the recoverer must only look at submissions the broker never confirmed")
			}
			return listedRows(listed...), nil
		},
	}, q)
	return r, &execSQL
}

func TestRecoverStalePending_RepublishesTheUnqueuedSubmissions(t *testing.T) {
	q := &mockAdapterQueue{}
	r, _ := newRecoverer(q, "a", "b")

	requeued, failed, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 2, requeued)
	assert.Equal(t, 0, failed)
	require.Len(t, q.published, 2)
	assert.Equal(t, "a", q.published[0].SubmissionID)
	assert.Equal(t, testProblemID, q.published[0].Metadata.ProblemID)
}

// Exhausted submissions are failed by their own statement, so one that keeps
// going missing is not retried forever.
func TestRecoverStalePending_FailsTheExhaustedOnes(t *testing.T) {
	r, execSQL := newRecoverer(&mockAdapterQueue{})

	_, _, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	require.NotEmpty(t, *execSQL)
	assert.Contains(t, (*execSQL)[0], "requeue_count >= $2")
	assert.Contains(t, (*execSQL)[0], "SYSTEM_ERROR")
	assert.Contains(t, (*execSQL)[0], "queued_at IS NULL")
}

// A broker that is down fails every publish; trying all 100 would only pile up
// timeouts, so the sweep stops at the first failure and the next one retries.
func TestRecoverStalePending_StopsAtTheFirstPublishFailure(t *testing.T) {
	q := &mockAdapterQueue{failOn: map[string]bool{"a": true}}
	r, _ := newRecoverer(q, "a", "b", "c")

	requeued, _, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, requeued)
	assert.Empty(t, q.published)
}

func TestRecoverStalePending_DBErrorReturnsAnError(t *testing.T) {
	q := &mockAdapterQueue{}
	r := NewStalePendingRecoverer(&mockQuerier{
		execFn: func(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db down")
		},
	}, q)

	_, _, err := r.RecoverStalePending(context.Background(), time.Now())
	require.Error(t, err)
	assert.Empty(t, q.published)
}
