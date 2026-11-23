package queue

import (
	"errors"
	"sync"
)

var errHandlerFailed = errors.New("handler failed")

type ackCall struct {
	method   string
	multiple bool
	requeue  bool
}

type mockAcknowledger struct {
	mu    sync.Mutex
	calls []ackCall
}

func (m *mockAcknowledger) Ack(_ uint64, multiple bool) error {
	m.record(ackCall{method: "ack", multiple: multiple})
	return nil
}

func (m *mockAcknowledger) Nack(_ uint64, multiple, requeue bool) error {
	m.record(ackCall{method: "nack", multiple: multiple, requeue: requeue})
	return nil
}

func (m *mockAcknowledger) Reject(_ uint64, requeue bool) error {
	m.record(ackCall{method: "reject", requeue: requeue})
	return nil
}

func (m *mockAcknowledger) record(c ackCall) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, c)
}
