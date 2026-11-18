package judge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
)

// fakeSource is storage with a download counter and a gate to hold downloads open.
type fakeSource struct {
	mu      sync.Mutex
	objects map[string][]byte
	reads   map[string]int
	// reported overrides the size the object claims, to simulate a stale or lying one.
	reported map[string]int64
	readErr  error
	gate     chan struct{}
}

func newFakeSource() *fakeSource {
	return &fakeSource{objects: map[string][]byte{}, reads: map[string]int{}, reported: map[string]int64{}}
}

func (f *fakeSource) objectSize(_ context.Context, key string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[key]
	if !ok {
		return 0, storage.ErrObjectNotExist
	}
	if size, ok := f.reported[key]; ok {
		return size, nil
	}
	return int64(len(data)), nil
}

func (f *fakeSource) readObject(ctx context.Context, key string) (io.ReadCloser, error) {
	f.mu.Lock()
	f.reads[key]++
	data, ok := f.objects[key]
	gate, readErr := f.gate, f.readErr
	f.mu.Unlock()

	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if readErr != nil {
		return nil, readErr
	}
	if !ok {
		return nil, storage.ErrObjectNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeSource) readCount(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads[key]
}

// archive makes a valid archive of a predictable size: same-length contents give same-length archives.
func archive(t *testing.T, fill string) []byte {
	t.Helper()
	return zipBytes(t, map[string]string{
		"data/secret/1.in":  strings.Repeat(fill, 300),
		"data/secret/1.ans": strings.Repeat(fill, 20),
	}, true)
}

func newCache(t *testing.T, src *fakeSource, budget int64) *testDataCache {
	t.Helper()
	limits := testLimits()
	limits.MaxZipBytes = min(4096, budget) // a budget below the limit is refused by the constructor
	cache, err := newTestDataCache(TestDataConfig{
		Dir:         t.TempDir(),
		BudgetBytes: budget,
		Limits:      limits,
		ReserveWait: 300 * time.Millisecond,
	}, src)
	if err != nil {
		t.Fatalf("newTestDataCache: %v", err)
	}
	return cache
}

func (c *testDataCache) usedBytes() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used
}

func TestTestDataCache_SecondAcquireReusesTheDownload(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	cache := newCache(t, src, 1<<20)

	first, err := cache.acquire(context.Background(), "a.zip")
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	cache.release(first)
	second, err := cache.acquire(context.Background(), "a.zip")
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	defer cache.release(second)

	if got := src.readCount("a.zip"); got != 1 {
		t.Errorf("downloaded %d times, want once", got)
	}
	if first.path != second.path {
		t.Errorf("the archive moved: %s then %s", first.path, second.path)
	}
}

func TestTestDataCache_ConcurrentAcquiresOfOneKeyShareOneDownload(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.gate = make(chan struct{})
	cache := newCache(t, src, 1<<20)

	const callers = 8
	entries := make([]*cacheEntry, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entries[i], errs[i] = cache.acquire(context.Background(), "a.zip")
		}()
	}
	time.Sleep(100 * time.Millisecond) // let every caller reach the held download
	close(src.gate)
	wg.Wait()

	for i := range entries {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if entries[i] != entries[0] {
			t.Errorf("caller %d got another entry", i)
		}
	}
	if got := src.readCount("a.zip"); got != 1 {
		t.Errorf("downloaded %d times for %d callers, want once", got, callers)
	}
	if got := cache.usedBytes(); got != int64(len(src.objects["a.zip"])) {
		t.Errorf("used = %d, want the size of one archive (%d): the shared download was counted more than once", got, len(src.objects["a.zip"]))
	}
	for _, e := range entries {
		cache.release(e)
	}
}

func TestTestDataCache_FailedDownloadFailsEveryWaiterAndIsNotKept(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.gate = make(chan struct{})
	src.readErr = errors.New("storage down")
	cache := newCache(t, src, 1<<20)

	errs := make([]error, 4)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = cache.acquire(context.Background(), "a.zip")
		}()
	}
	time.Sleep(100 * time.Millisecond)
	close(src.gate)
	wg.Wait()

	for i, err := range errs {
		if err == nil {
			t.Errorf("caller %d: expected the failure to reach every waiter", i)
		}
	}
	if got := cache.usedBytes(); got != 0 {
		t.Errorf("used = %d after a failed download, want 0: the reservation leaked", got)
	}

	// Not kept: once storage is back the next caller downloads again and succeeds.
	src.mu.Lock()
	src.readErr = nil
	src.mu.Unlock()
	entry, err := cache.acquire(context.Background(), "a.zip")
	if err != nil {
		t.Fatalf("expected a retry to succeed once storage is back, got %v", err)
	}
	cache.release(entry)
}

func TestTestDataCache_InvalidArchiveIsRefusedAndLeavesNothing(t *testing.T) {
	src := newFakeSource()
	src.objects["bad.zip"] = bytes.Repeat([]byte("not a zip "), 20)
	cache := newCache(t, src, 1<<20)

	_, err := cache.acquire(context.Background(), "bad.zip")

	if !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
	if got := cache.usedBytes(); got != 0 {
		t.Errorf("used = %d, want 0", got)
	}
	if files, _ := os.ReadDir(cache.dir); len(files) != 0 {
		t.Errorf("a refused archive left %d files on disk", len(files))
	}
}

func TestTestDataCache_ObjectOverTheLimitIsRefusedWithoutDownloading(t *testing.T) {
	src := newFakeSource()
	src.objects["big.zip"] = bytes.Repeat([]byte("x"), 5000) // the test limit is 4096
	cache := newCache(t, src, 1<<20)

	_, err := cache.acquire(context.Background(), "big.zip")

	if !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
	if got := src.readCount("big.zip"); got != 0 {
		t.Errorf("downloaded %d times: an object over the limit must not be read at all", got)
	}
}

func TestTestDataCache_ObjectThatDoesNotMatchItsReportedSizeIsRefused(t *testing.T) {
	src := newFakeSource()
	data := archive(t, "a")
	src.objects["a.zip"] = data
	src.reported["a.zip"] = int64(len(data)) - 10 // storage says smaller than what arrives
	cache := newCache(t, src, 1<<20)

	_, err := cache.acquire(context.Background(), "a.zip")

	if !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
	if got := cache.usedBytes(); got != 0 {
		t.Errorf("used = %d, want 0", got)
	}
}

func TestTestDataCache_MissingObjectIsReportedAsNotFound(t *testing.T) {
	cache := newCache(t, newFakeSource(), 1<<20)

	_, err := cache.acquire(context.Background(), "nope.zip")

	if !errors.Is(err, storage.ErrObjectNotExist) {
		t.Fatalf("expected storage.ErrObjectNotExist, got %v", err)
	}
}

func TestTestDataCache_EvictsTheLeastRecentlyUsedIdleArchive(t *testing.T) {
	src := newFakeSource()
	for _, k := range []string{"a", "b", "c"} {
		src.objects[k+".zip"] = archive(t, k)
	}
	size := int64(len(src.objects["a.zip"]))
	cache := newCache(t, src, 2*size+10) // room for two

	for _, k := range []string{"a.zip", "b.zip"} {
		e, err := cache.acquire(context.Background(), k)
		if err != nil {
			t.Fatalf("acquire %s: %v", k, err)
		}
		cache.release(e)
		time.Sleep(5 * time.Millisecond) // distinct last-used times
	}
	e, err := cache.acquire(context.Background(), "c.zip")
	if err != nil {
		t.Fatalf("acquire c.zip: %v", err)
	}
	cache.release(e)

	for _, k := range []string{"b.zip", "c.zip"} {
		e, err := cache.acquire(context.Background(), k)
		if err != nil {
			t.Fatalf("re-acquire %s: %v", k, err)
		}
		cache.release(e)
	}
	if got := src.readCount("b.zip"); got != 1 {
		t.Errorf("b.zip downloaded %d times, want 1: it was recent and must have stayed", got)
	}
	e, _ = cache.acquire(context.Background(), "a.zip")
	cache.release(e)
	if got := src.readCount("a.zip"); got != 2 {
		t.Errorf("a.zip downloaded %d times, want 2: it was the oldest and must have been evicted", got)
	}
	if got := cache.usedBytes(); got > 2*size+10 {
		t.Errorf("used = %d, over the budget %d", got, 2*size+10)
	}
}

func TestTestDataCache_NeverEvictsAnArchiveSomeoneHolds_AndWaitsForRoom(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.objects["b.zip"] = archive(t, "b")
	size := int64(len(src.objects["a.zip"]))
	cache := newCache(t, src, size+10) // room for one
	cache.maxWait = 5 * time.Second

	held, err := cache.acquire(context.Background(), "a.zip")
	if err != nil {
		t.Fatalf("acquire a.zip: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		e, err := cache.acquire(context.Background(), "b.zip")
		if err == nil {
			cache.release(e)
		}
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("b.zip returned (%v) while a.zip was held and the budget fits one", err)
	case <-time.After(150 * time.Millisecond):
	}
	if _, err := os.Stat(held.path); err != nil {
		t.Fatalf("the held archive was removed from disk: %v", err)
	}

	cache.release(held)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("b.zip should proceed once a.zip is released, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("b.zip never proceeded after a.zip was released")
	}
}

func TestTestDataCache_WaitForRoomTimesOutAsBusy(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.objects["b.zip"] = archive(t, "b")
	size := int64(len(src.objects["a.zip"]))
	cache := newCache(t, src, size+10) // maxWait is 300ms

	held, err := cache.acquire(context.Background(), "a.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.release(held)

	_, err = cache.acquire(context.Background(), "b.zip")

	if !errors.Is(err, errTestDataBusy) {
		t.Fatalf("expected errTestDataBusy, got %v", err)
	}
	if got := cache.usedBytes(); got != size {
		t.Errorf("used = %d, want %d: the waiter reserved nothing", got, size)
	}
}

func TestTestDataCache_WaitForRoomEndsWhenTheContextIsCancelled(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.objects["b.zip"] = archive(t, "b")
	size := int64(len(src.objects["a.zip"]))
	cache := newCache(t, src, size+10)
	cache.maxWait = 5 * time.Second

	held, _ := cache.acquire(context.Background(), "a.zip")
	defer cache.release(held)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := cache.acquire(ctx, "b.zip")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the context's error, got %v", err)
	}
}

func TestTestDataCache_ACancelledWaiterOfASharedDownloadDoesNotBreakTheOthers(t *testing.T) {
	src := newFakeSource()
	src.objects["a.zip"] = archive(t, "a")
	src.gate = make(chan struct{})
	cache := newCache(t, src, 1<<20)

	owner := make(chan error, 1)
	go func() {
		e, err := cache.acquire(context.Background(), "a.zip")
		if err == nil {
			cache.release(e)
		}
		owner <- err
	}()
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := cache.acquire(ctx, "a.zip"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the waiter to give up, got %v", err)
	}

	close(src.gate)
	if err := <-owner; err != nil {
		t.Fatalf("the download's owner must be unaffected by a waiter leaving, got %v", err)
	}
}

func TestNewTestDataCache_WipesLeftoversAndKeepsTheDirectory(t *testing.T) {
	dir := t.TempDir()
	leftover := filepath.Join(dir, "deadbeef.zip.tmp")
	if err := os.WriteFile(leftover, []byte("half a download"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := newTestDataCache(TestDataConfig{Dir: dir, BudgetBytes: 1 << 20, Limits: testLimits()}, newFakeSource())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Errorf("a leftover of a dead process must be removed, stat err = %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the directory itself (a mount point in production) must stay: %v", err)
	}
}

func TestNewTestDataCache_BudgetUnderTheLargestArchiveIsRefused(t *testing.T) {
	limits := testLimits()
	limits.MaxZipBytes = 1000

	_, err := newTestDataCache(TestDataConfig{Dir: t.TempDir(), BudgetBytes: 999, Limits: limits}, newFakeSource())

	if err == nil {
		t.Fatal("a budget that cannot hold one archive must be refused")
	}
}

// What the worker relies on: judgings of different problems, in the numbers the pod runs
// them, all get through a budget that fits them and none waits on another.
func TestTestDataCache_ConcurrentJudgingsOfDifferentProblemsAllProceed(t *testing.T) {
	const judgings = 5
	src := newFakeSource()
	var size int64
	for i := 0; i < judgings; i++ {
		src.objects[fmt.Sprintf("p%d.zip", i)] = archive(t, fmt.Sprintf("%d", i))
		size = int64(len(src.objects["p0.zip"]))
	}
	cache := newCache(t, src, judgings*size)
	cache.maxWait = 5 * time.Second

	var wg sync.WaitGroup
	errs := make(chan error, judgings)
	for i := 0; i < judgings; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := cache.acquire(context.Background(), fmt.Sprintf("p%d.zip", i))
			if err != nil {
				errs <- err
				return
			}
			time.Sleep(20 * time.Millisecond)
			cache.release(e)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("a judging failed: %v", err)
	}
}
