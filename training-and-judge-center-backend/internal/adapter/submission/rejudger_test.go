package submission

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// callLog records the order of database and queue calls across both mocks.
type callLog struct{ calls []string }

func (l *callLog) add(c string) { l.calls = append(l.calls, c) }

type mockAdapterQueue struct {
	log       *callLog
	published []appSubmission.SubmissionQueueMessage
	failOn    map[string]bool
}

func (m *mockAdapterQueue) Publish(_ context.Context, msg appSubmission.SubmissionQueueMessage) error {
	if m.log != nil {
		m.log.add("publish:" + msg.SubmissionID)
	}
	if m.failOn[msg.SubmissionID] {
		return errors.New("broker down")
	}
	m.published = append(m.published, msg)
	return nil
}

// resetRows answers the reset statement with one previous-state row per id.
func resetRows(ids ...string) pgx.Rows {
	rows := &mockRows{}
	for _, id := range ids {
		rows.scanFns = append(rows.scanFns, func(dest ...any) error {
			*(dest[0].(*string)) = id
			*(dest[1].(*string)) = "WRONG_ANSWER"
			return nil
		})
	}
	return rows
}

// newRecordingRejudger returns a rejudger whose reset statement matches every
// id passed in resettable, and whose calls (reset, restore, publish) go to log.
func newRecordingRejudger(log *callLog, q *mockAdapterQueue, resettable ...string) *Rejudger {
	return NewRejudger(&mockQuerier{
		queryFn: func(_ context.Context, sql string, _ ...interface{}) (pgx.Rows, error) {
			log.add("reset")
			return resetRows(resettable...), nil
		},
		execFn: func(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "SET status = $2") {
				log.add("restore:" + args[0].(string))
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}, q)
}

func TestRejudgeBatch_PublishesWithRejudgePriority(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	r := newRecordingRejudger(log, q, testSubID)

	subs := []appProblem.SubmissionRejudgeInfo{
		{ID: testSubID, UserID: testUserID, ContestID: nil, Language: "cpp20"},
	}
	count, err := r.RejudgeBatch(context.Background(), subs, testProblemID, testNow)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, q.published, 1)
	assert.Equal(t, appSubmission.QueuePriorityRejudge, q.published[0].Priority)
}

// The worker drops a message whose submission is not PENDING, so the reset has
// to be done before any publish.
func TestRejudgeBatch_ResetsEverythingBeforeTheFirstPublish(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	r := newRecordingRejudger(log, q, "a", "b")

	_, err := r.RejudgeBatch(context.Background(), []appProblem.SubmissionRejudgeInfo{{ID: "a"}, {ID: "b"}}, testProblemID, testNow)
	require.NoError(t, err)
	assert.Equal(t, []string{"reset", "publish:a", "publish:b"}, log.calls)
}

func TestRejudgeBatch_RestoresTheSubmissionsWhosePublishFailed(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log, failOn: map[string]bool{"b": true}}
	r := newRecordingRejudger(log, q, "a", "b")

	count, err := r.RejudgeBatch(context.Background(), []appProblem.SubmissionRejudgeInfo{{ID: "a"}, {ID: "b"}}, testProblemID, testNow)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, []string{"reset", "publish:a", "publish:b", "restore:b"}, log.calls)
}

func TestRejudgeBatch_SkipsSubmissionsTheResetDidNotTouch(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	// "b" is already RUNNING, so the reset does not return it.
	r := newRecordingRejudger(log, q, "a")

	count, err := r.RejudgeBatch(context.Background(), []appProblem.SubmissionRejudgeInfo{{ID: "a"}, {ID: "b"}}, testProblemID, testNow)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, []string{"reset", "publish:a"}, log.calls)
}

func TestRejudgeBatch_ResetFailureQueuesNothing(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	r := NewRejudger(&mockQuerier{
		queryFn: func(context.Context, string, ...interface{}) (pgx.Rows, error) {
			return nil, errors.New("db down")
		},
	}, q)

	_, err := r.RejudgeBatch(context.Background(), []appProblem.SubmissionRejudgeInfo{{ID: "a"}}, testProblemID, testNow)
	require.Error(t, err)
	assert.Empty(t, log.calls)
}

func TestRejudgeByID_ResetsBeforePublishing(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	r := newRecordingRejudger(log, q, testSubID)

	require.NoError(t, r.RejudgeByID(context.Background(), testSubID, testProblemID, testUserID, nil, "cpp20", testNow))
	assert.Equal(t, []string{"reset", "publish:" + testSubID}, log.calls)
}

func TestRejudgeByID_PublishFailureRestoresTheSubmissionAndReturnsTheError(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log, failOn: map[string]bool{testSubID: true}}
	r := newRecordingRejudger(log, q, testSubID)

	err := r.RejudgeByID(context.Background(), testSubID, testProblemID, testUserID, nil, "cpp20", testNow)
	require.Error(t, err)
	assert.Equal(t, []string{"reset", "publish:" + testSubID, "restore:" + testSubID}, log.calls)
}

func TestRejudgeByID_AlreadyInProgressPublishesNothing(t *testing.T) {
	log := &callLog{}
	q := &mockAdapterQueue{log: log}
	r := newRecordingRejudger(log, q) // reset touches no row

	require.NoError(t, r.RejudgeByID(context.Background(), testSubID, testProblemID, testUserID, nil, "cpp20", testNow))
	assert.Equal(t, []string{"reset"}, log.calls)
}
