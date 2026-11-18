package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestLimitConcurrency_NeverRunsMoreThanMaxAtOnce(t *testing.T) {
	const max = 3
	var mu sync.Mutex
	running, peak := 0, 0
	release := make(chan struct{})

	h := LimitConcurrency(max, 2*time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		running++
		if running > peak {
			peak = running
		}
		mu.Unlock()
		<-release
		mu.Lock()
		running--
		mu.Unlock()
	}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
		}()
	}

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if running != max {
		t.Errorf("running = %d while saturated, want %d", running, max)
	}
	mu.Unlock()

	close(release)
	wg.Wait()
	if peak > max {
		t.Errorf("peak concurrency %d exceeded the limit %d", peak, max)
	}
}

func TestLimitConcurrency_NoSlotWithinWait_Returns503WithRetryAfter(t *testing.T) {
	block := make(chan struct{})
	h := LimitConcurrency(1, 50*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))

	go h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	time.Sleep(20 * time.Millisecond)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	close(block)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After must be set")
	}
}

func TestLimitConcurrency_SlotIsFreedWhenAHandlerPanics(t *testing.T) {
	calls := 0
	h := LimitConcurrency(1, 50*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			panic("boom")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	func() {
		defer func() { _ = recover() }()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	}()

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d: the slot of the panicked handler was never released", w.Code)
	}
}

func TestLimitConcurrency_ClientGoneWhileWaiting_DoesNotRunTheHandler(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	ran := false
	h := LimitConcurrency(1, 5*time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Second") != "" {
			ran = true
			return
		}
		<-block
	}))
	go h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	r.Header.Set("X-Second", "1")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if ran {
		t.Error("a request whose client is gone must not take a slot")
	}
}
