package queue

import (
	"fmt"
	"log/slog"
	"time"
)

const (
	connectInitialDelay = time.Second
	connectMaxDelay     = 5 * time.Second
)

// ConnectRabbitMQQueue keeps dialing until the broker answers or timeout runs
// out: a broker that reports healthy can still refuse AMQP for a few seconds.
func ConnectRabbitMQQueue(url string, timeout time.Duration) (*RabbitMQQueue, error) {
	return connectWithRetry(url, timeout, connectInitialDelay)
}

func connectWithRetry(url string, timeout, delay time.Duration) (*RabbitMQQueue, error) {
	deadline := time.Now().Add(timeout)
	for attempt := 1; ; attempt++ {
		q, err := NewRabbitMQQueue(url)
		if err == nil {
			return q, nil
		}
		if time.Now().Add(delay).After(deadline) {
			return nil, fmt.Errorf("rabbitmq: gave up after %d attempts: %w", attempt, err)
		}
		slog.Warn("rabbitmq not ready, retrying", "attempt", attempt, "retry_in", delay, "error", err)
		time.Sleep(delay)
		delay = min(delay*2, connectMaxDelay)
	}
}
