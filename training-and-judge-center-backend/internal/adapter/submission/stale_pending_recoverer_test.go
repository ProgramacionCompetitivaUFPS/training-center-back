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

func claimedRows(ids ...string) pgx.Rows {
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

func newRecoverer(q appSubmission.SubmissionQueue, failed int64, claimed ...string) (*StalePendingRecoverer, *[]string) {
	var execSQL []string
	r := NewStalePendingRecoverer(&mockQuerier{
		execFn: func(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
			execSQL = append(execSQL, sql)
			return pgconn.NewCommandTag("UPDATE " + string(rune('0'+failed))), nil
		},
		queryFn: func(context.Context, string, ...interface{}) (pgx.Rows, error) {
			return claimedRows(claimed...), nil
		},
	}, q)
	return r, &execSQL
}

func TestRecoverStalePending_RepublishesTheClaimedSubmissions(t *testing.T) {
	q := &mockAdapterQueue{}
	r, _ := newRecoverer(q, 0, "a", "b")

	requeued, failed, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 2, requeued)
	assert.Equal(t, 0, failed)
	require.Len(t, q.published, 2)
	assert.Equal(t, "a", q.published[0].SubmissionID)
	assert.Equal(t, testProblemID, q.published[0].Metadata.ProblemID)
}

// The exhausted ones are failed by their own statement, so a submission that
// keeps getting lost is not retried forever.
func TestRecoverStalePending_FailsTheExhaustedOnes(t *testing.T) {
	r, execSQL := newRecoverer(&mockAdapterQueue{}, 2)

	requeued, failed, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, requeued)
	assert.Equal(t, 2, failed)
	require.Len(t, *execSQL, 1)
	assert.True(t, strings.Contains((*execSQL)[0], "requeue_count >= $2") && strings.Contains((*execSQL)[0], "SYSTEM_ERROR"))
}

func TestRecoverStalePending_PublishFailureIsNotRetriedWithinTheSweep(t *testing.T) {
	q := &mockAdapterQueue{failOn: map[string]bool{"a": true}}
	r, _ := newRecoverer(q, 0, "a", "b")

	requeued, _, err := r.RecoverStalePending(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, requeued)
	require.Len(t, q.published, 1)
	assert.Equal(t, "b", q.published[0].SubmissionID)
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
