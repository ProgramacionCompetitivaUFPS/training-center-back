package queue

import (
	"testing"
	"time"
)

func TestConnectWithRetry_GivesUpAfterTheTimeout(t *testing.T) {
	start := time.Now()
	// Port 1 refuses connections immediately.
	q, err := connectWithRetry("amqp://guest:guest@127.0.0.1:1/", 300*time.Millisecond, 50*time.Millisecond)
	if err == nil {
		q.Close()
		t.Fatal("expected an error when the broker never answers")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v, the timeout should bound the retries", elapsed)
	}
}
