package submission

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

type mockAdapterQueue struct {
	published []appSubmission.SubmissionQueueMessage
}

func (m *mockAdapterQueue) Publish(_ context.Context, msg appSubmission.SubmissionQueueMessage) error {
	m.published = append(m.published, msg)
	return nil
}

func TestRejudgeBatch_PublishesWithRejudgePriority(t *testing.T) {
	q := &mockAdapterQueue{}
	r := NewRejudger(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}, q)

	subs := []appProblem.SubmissionRejudgeInfo{
		{ID: testSubID, UserID: testUserID, ContestID: nil, Language: "cpp20"},
	}
	count, err := r.RejudgeBatch(context.Background(), subs, testProblemID, testNow)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, q.published, 1)
	assert.Equal(t, appSubmission.QueuePriorityRejudge, q.published[0].Priority)
}
