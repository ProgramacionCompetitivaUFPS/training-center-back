package judge

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func judgeLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func judgingUseCase() *JudgeSubmissionUseCase {
	return newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{}, &mockSourceCodeDownloader{}, &mockProblemProvider{},
		&mockTestCaseProvider{}, &mockExecutor{}, &mockOutputChecker{},
	)
}

func TestJudgeSubmission_LogsTheVerdictTheQueueWaitAndTheTime(t *testing.T) {
	logs := judgeLogs(t)
	uc := judgingUseCase()
	// The clock reads one second later each time it is asked, so the judging takes some positive time.
	// The exact wait depends on how many readings come first, so only that it is reported is checked here;
	// the next test pins its value.
	clock := testNow
	uc.now = func() time.Time { clock = clock.Add(time.Second); return clock }

	enqueuedAt := testNow.Add(-7 * time.Second)
	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID, EnqueuedAt: enqueuedAt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := logs.String()
	if got := strings.Count(out, "submission judged"); got != 1 {
		t.Fatalf("logged %d lines, want exactly one per judging:\n%s", got, out)
	}
	for _, field := range []string{"verdict=ACCEPTED", "language=cpp20", "queue_wait=", "judged_in="} {
		if !strings.Contains(out, field) {
			t.Errorf("the line must carry %q:\n%s", field, out)
		}
	}
}

func TestJudgeSubmission_QueueWaitIsMeasuredFromWhenTheMessageWasPublished(t *testing.T) {
	logs := judgeLogs(t)
	uc := judgingUseCase()
	uc.now = func() time.Time { return testNow } // a clock that does not move: the wait is then exactly the gap

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID, EnqueuedAt: testNow.Add(-7 * time.Second)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(logs.String(), "queue_wait=7s") {
		t.Errorf("expected queue_wait=7s:\n%s", logs)
	}
	if !strings.Contains(logs.String(), "judged_in=0s") {
		t.Errorf("expected judged_in=0s from a clock that does not move:\n%s", logs)
	}
}

func TestJudgeSubmission_WithoutAnEnqueueTimeItLogsNoQueueWait(t *testing.T) {
	logs := judgeLogs(t)

	if err := judgingUseCase().Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(logs.String(), "queue_wait") {
		t.Errorf("an unknown enqueue time must not be reported as a wait:\n%s", logs)
	}
}

// Only a submission this worker took is logged: one that was skipped or lost the claim was judged
// somewhere else, or not at all, and a line for it would be a second record of the same judging.
func TestJudgeSubmission_ASubmissionItDidNotTakeLogsNoJudging(t *testing.T) {
	t.Run("not pending", func(t *testing.T) {
		logs := judgeLogs(t)
		running := pendingSubmission()
		_ = running.Start(testNow)
		uc := newJudgeSubmissionUseCase(
			&mockSubmissionUpdater{getByIDFn: func(context.Context, submission.SubmissionID) (*submission.Submission, error) { return running, nil }},
			&mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, &mockExecutor{}, &mockOutputChecker{},
		)

		_ = uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID})

		if strings.Contains(logs.String(), "submission judged") {
			t.Errorf("a submission that was skipped must not be logged as judged:\n%s", logs)
		}
	})
	t.Run("lost the claim", func(t *testing.T) {
		logs := judgeLogs(t)
		uc := newJudgeSubmissionUseCase(
			&mockSubmissionUpdater{claimFn: func(context.Context, submission.SubmissionID) (bool, error) { return false, nil }},
			&mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, &mockExecutor{}, &mockOutputChecker{},
		)

		_ = uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID})

		if strings.Contains(logs.String(), "submission judged") {
			t.Errorf("a submission another worker took must not be logged as judged here:\n%s", logs)
		}
	})
}
