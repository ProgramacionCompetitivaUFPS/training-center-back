package judge

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// defaultReserveWait is how long a judging waits for disk before giving up with a
	// retryable error. With the budget invariant it only ever waits for another judging
	// to finish, which is far shorter.
	defaultReserveWait = 2 * time.Minute
	// downloadTimeout bounds getting one archive in (download and extraction), so a stalled
	// transfer cannot hold its slot, and the disk it reserved, for good.
	downloadTimeout = 5 * time.Minute
)

// errTestDataBusy means no disk became free within the wait. Retrying later can succeed.
var errTestDataBusy = errors.New("no disk available for the test case archive")

// TestDataConfig sizes the test data cache.
type TestDataConfig struct {
	// Dir holds the cached problems. Its contents are wiped at startup.
	Dir string
	// BudgetBytes is the most disk the cache may take, counting an archive while it is being
	// extracted. See validateTestDataBudget for what it has to cover.
	BudgetBytes int64
	Limits      TestDataLimits
	// ReserveWait defaults to defaultReserveWait.
	ReserveWait time.Duration
}

type cacheEntry struct {
	key  string
	dir  string
	size int64 // the archive, held on disk only while the problem is being extracted
	// reserved is what this entry counts against the budget: the archive and the extraction while
	// loading, and the extracted cases alone once it is ready.
	reserved int64
	cases    []cachedCase
	refs     int
	lastUsed time.Time
	ready    chan struct{}
	err      error
}

// testDataCache keeps test cases on disk, extracted, so a judging copies its cases from local
// files and neither decompresses anything nor holds a problem's data in memory.
//
// The key is the archive's object key. An upload writes under a fresh UUID and the old key is
// deleted, so a key never changes content: there is nothing to invalidate.
type testDataCache struct {
	dir     string
	budget  int64
	limits  TestDataLimits
	source  sizedObjectReader
	maxWait time.Duration
	now     func() time.Time

	mu      sync.Mutex
	used    int64
	entries map[string]*cacheEntry
	// changed is closed, and replaced, whenever disk may have become available.
	changed chan struct{}
}

func newTestDataCache(cfg TestDataConfig, source sizedObjectReader) (*testDataCache, error) {
	if cfg.BudgetBytes < cfg.Limits.MaxZipBytes+cfg.Limits.MaxExtractedBytes {
		return nil, fmt.Errorf("test data budget %d cannot hold one archive (%d) and its extraction (%d)",
			cfg.BudgetBytes, cfg.Limits.MaxZipBytes, cfg.Limits.MaxExtractedBytes)
	}
	if err := resetDir(cfg.Dir); err != nil {
		return nil, fmt.Errorf("prepare test data dir %s: %w", cfg.Dir, err)
	}
	wait := cfg.ReserveWait
	if wait <= 0 {
		wait = defaultReserveWait
	}
	return &testDataCache{
		dir:     cfg.Dir,
		budget:  cfg.BudgetBytes,
		limits:  cfg.Limits,
		source:  source,
		maxWait: wait,
		now:     time.Now,
		entries: make(map[string]*cacheEntry),
		changed: make(chan struct{}),
	}, nil
}

// resetDir leaves dir empty and existing. Anything in it is a leftover of a process that
// died, and the directory itself may be a mount point, so only its contents go.
func resetDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := os.RemoveAll(filepath.Join(dir, item.Name())); err != nil {
			return err
		}
	}
	return nil
}

// acquire returns the problem for key with its reference count raised; the caller must
// release it. Concurrent callers for one key share a single download, and a download that
// fails fails every caller waiting on it and is never kept.
func (c *testDataCache) acquire(ctx context.Context, key string) (*cacheEntry, error) {
	deadline := time.NewTimer(c.maxWait)
	defer deadline.Stop()

	size := int64(-1)
	parked := &roomWait{cache: c, what: "an archive", key: key}
	for {
		c.mu.Lock()
		if e, ok := c.entries[key]; ok {
			e.refs++
			e.lastUsed = c.now()
			c.mu.Unlock()
			return c.await(ctx, e)
		}
		if size < 0 {
			// Asked outside the lock: it is a network call. Looping re-checks the map after.
			c.mu.Unlock()
			var err error
			if size, err = c.source.objectSize(ctx, key); err != nil {
				return nil, err
			}
			parked.need = size
			if size > c.limits.MaxZipBytes {
				return nil, fmt.Errorf("%w: archive is %d bytes, over the %d byte limit", errInvalidTestData, size, c.limits.MaxZipBytes)
			}
			continue
		}

		reserved, wait := c.reserveLocked(ctx, key, size)
		used := c.used
		c.mu.Unlock()
		if reserved != nil {
			parked.end(ctx)
			return c.fill(ctx, key, reserved)
		}
		parked.begin(ctx, used)
		select {
		case <-wait:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			parked.giveUp(ctx)
			return nil, errTestDataBusy
		}
	}
}

// reserveLocked takes the archive's size of the budget for a new entry, evicting problems nobody
// holds, least recently used first. When every problem is held it returns the channel to wait on
// instead. Caller holds c.mu.
func (c *testDataCache) reserveLocked(ctx context.Context, key string, size int64) (*cacheEntry, <-chan struct{}) {
	if wait := c.makeRoomLocked(ctx, size); wait != nil {
		return nil, wait
	}
	sum := sha256.Sum256([]byte(key))
	e := &cacheEntry{
		key:      key,
		dir:      filepath.Join(c.dir, hex.EncodeToString(sum[:])),
		size:     size,
		reserved: size,
		refs:     1,
		lastUsed: c.now(),
		ready:    make(chan struct{}),
	}
	c.entries[key] = e
	c.used += size
	return e, nil
}

// makeRoomLocked evicts until need more bytes fit, or returns the channel to wait on when what
// is left is held. Caller holds c.mu.
func (c *testDataCache) makeRoomLocked(ctx context.Context, need int64) <-chan struct{} {
	for c.used+need > c.budget {
		victim := c.oldestIdleLocked()
		if victim == nil {
			return c.changed
		}
		c.dropLocked(ctx, victim)
	}
	return nil
}

func (c *testDataCache) oldestIdleLocked() *cacheEntry {
	var oldest *cacheEntry
	for _, e := range c.entries {
		if e.refs > 0 {
			continue
		}
		if oldest == nil || e.lastUsed.Before(oldest.lastUsed) {
			oldest = e
		}
	}
	return oldest
}

// dropLocked evicts a problem nobody holds. It is logged because a problem that is evicted and then asked
// for again costs a download and an extraction: repeated evictions are how an undersized budget shows.
func (c *testDataCache) dropLocked(ctx context.Context, e *cacheEntry) {
	delete(c.entries, e.key)
	_ = os.RemoveAll(e.dir)
	c.used -= e.reserved
	slog.InfoContext(ctx, "test_data_cache: evicted an idle problem to make room",
		"key", e.key, "freed_bytes", e.reserved, "idle_for", c.now().Sub(e.lastUsed).Round(time.Second), "used_bytes", c.used, "budget_bytes", c.budget)
}

// await waits for the download another caller started.
func (c *testDataCache) await(ctx context.Context, e *cacheEntry) (*cacheEntry, error) {
	select {
	case <-e.ready:
	case <-ctx.Done():
		c.release(e)
		return nil, ctx.Err()
	}
	if e.err != nil {
		c.release(e)
		return nil, e.err
	}
	return e, nil
}

// fill gets the problem in on behalf of this caller, which reserved disk for it, then publishes
// the outcome to whoever is waiting on it.
func (c *testDataCache) fill(ctx context.Context, key string, e *cacheEntry) (*cacheEntry, error) {
	err := c.populate(ctx, e)

	c.mu.Lock()
	if err != nil {
		e.err = err
		if c.entries[key] == e {
			delete(c.entries, key)
			c.used -= e.reserved
		}
	}
	close(e.ready)
	c.notifyLocked()
	c.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return e, nil
}

// populate downloads the archive, checks it against the limits, extracts its cases, and drops
// the archive. The extraction is built in a scratch directory and renamed into place, so what
// a judging can see is whole or absent.
func (c *testDataCache) populate(ctx context.Context, e *cacheEntry) error {
	started := time.Now()
	dctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	archivePath := e.dir + ".zip.tmp"
	scratch := e.dir + ".tmp"
	defer os.Remove(archivePath)
	defer os.RemoveAll(scratch)

	if err := c.download(dctx, e, archivePath); err != nil {
		return err
	}

	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("%w: %v", errInvalidTestData, err)
	}
	defer zr.Close()

	plan, err := planExtraction(&zr.Reader, c.limits)
	if err != nil {
		return err
	}
	if err := c.extend(dctx, e, plan.bytes); err != nil {
		return err
	}

	cases, err := extractTestCases(dctx, plan, scratch, e.dir)
	if err != nil {
		return err
	}
	if err := os.Rename(scratch, e.dir); err != nil {
		return err
	}
	e.cases = cases

	// Only the extracted cases stay: the archive goes with the deferred remove.
	c.mu.Lock()
	c.used -= e.size
	e.reserved -= e.size
	c.mu.Unlock()

	slog.InfoContext(ctx, "test_data_cache: problem cached",
		"key", e.key, "archive_bytes", e.size, "extracted_bytes", plan.bytes, "cases", len(cases), "took", time.Since(started).Round(time.Millisecond))
	return nil
}

func (c *testDataCache) download(ctx context.Context, e *cacheEntry, dst string) error {
	rc, err := c.source.readObject(ctx, e.key)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	// One byte past the size tells an object that grew from one that is exact.
	n, copyErr := io.Copy(f, io.LimitReader(rc, e.size+1))
	if err := errors.Join(copyErr, f.Close()); err != nil {
		return err
	}
	if n != e.size {
		return fmt.Errorf("%w: object has %d bytes, storage reported %d", errInvalidTestData, n, e.size)
	}
	return nil
}

// extend reserves the disk for the extraction, once the archive's directory says how much it is.
// The archive's own share is held meanwhile, so waiting here is the one place a fill holds disk
// while it waits for more: the budget invariant (validateTestDataBudget) is what guarantees that
// some fill can always go on.
func (c *testDataCache) extend(ctx context.Context, e *cacheEntry, extra int64) error {
	if e.size+extra > c.budget {
		return fmt.Errorf("%w: the archive and its extraction need %d bytes, over the %d byte budget", errInvalidTestData, e.size+extra, c.budget)
	}
	deadline := time.NewTimer(c.maxWait)
	defer deadline.Stop()

	parked := &roomWait{cache: c, what: "the extraction of a problem", key: e.key, need: extra}
	for {
		c.mu.Lock()
		wait := c.makeRoomLocked(ctx, extra)
		if wait == nil {
			c.used += extra
			e.reserved += extra
			c.mu.Unlock()
			parked.end(ctx)
			return nil
		}
		used := c.used
		c.mu.Unlock()
		parked.begin(ctx, used)

		select {
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			parked.giveUp(ctx)
			return errTestDataBusy
		}
	}
}

// roomWait logs a judging's wait for disk: once when it starts, and once when it ends or is given up.
// With the budget right a wait is rare, so each one is worth a line: it is how an undersized budget shows.
type roomWait struct {
	cache *testDataCache
	what  string
	key   string
	need  int64
	since time.Time
}

func (w *roomWait) begin(ctx context.Context, used int64) {
	if !w.since.IsZero() {
		return
	}
	w.since = w.cache.now()
	slog.WarnContext(ctx, "test_data_cache: waiting for disk",
		"for", w.what, "key", w.key, "need_bytes", w.need, "used_bytes", used, "budget_bytes", w.cache.budget)
}

func (w *roomWait) end(ctx context.Context) {
	if w.since.IsZero() {
		return
	}
	slog.InfoContext(ctx, "test_data_cache: got the disk it waited for",
		"for", w.what, "key", w.key, "waited", w.cache.now().Sub(w.since).Round(time.Millisecond))
}

func (w *roomWait) giveUp(ctx context.Context) {
	slog.ErrorContext(ctx, "test_data_cache: gave up waiting for disk",
		"for", w.what, "key", w.key, "need_bytes", w.need, "waited", w.cache.now().Sub(w.since).Round(time.Millisecond))
}

// release drops one reference. A problem nobody holds stays on disk until the space is needed,
// so the next judging of the same problem finds it there.
func (c *testDataCache) release(e *cacheEntry) {
	c.mu.Lock()
	e.refs--
	e.lastUsed = c.now()
	c.notifyLocked()
	c.mu.Unlock()
}

// notifyLocked wakes every judging waiting for disk. Closing and replacing a channel wakes all
// of them and loses no wake-up, which a single-slot signal would.
func (c *testDataCache) notifyLocked() {
	close(c.changed)
	c.changed = make(chan struct{})
}
