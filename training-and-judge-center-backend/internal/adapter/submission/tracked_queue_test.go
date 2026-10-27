package submission

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func TestTrackedQueue_MarksTheSubmissionQueuedAfterThePublish(t *testing.T) {
	log := &callLog{}
	inner := &mockAdapterQueue{log: log}
	tq := NewTrackedQueue(&mockQuerier{
		execFn: func(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "queued_at = now()") {
				log.add("mark:" + args[0].(string))
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}, inner)

	require.NoError(t, tq.Publish(context.Background(), appSubmission.SubmissionQueueMessage{SubmissionID: "a"}))
	assert.Equal(t, []string{"publish:a", "mark:a"}, log.calls)
}

func TestTrackedQueue_FailedPublishLeavesItUnmarked(t *testing.T) {
	log := &callLog{}
	inner := &mockAdapterQueue{log: log, failOn: map[string]bool{"a": true}}
	tq := NewTrackedQueue(&mockQuerier{
		execFn: func(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
			log.add("mark")
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}, inner)

	require.Error(t, tq.Publish(context.Background(), appSubmission.SubmissionQueueMessage{SubmissionID: "a"}))
	assert.Equal(t, []string{"publish:a"}, log.calls)
}
