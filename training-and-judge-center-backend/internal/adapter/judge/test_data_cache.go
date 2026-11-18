package judge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	// downloadTimeout bounds one archive's download, so a stalled transfer cannot hold
	// its slot (and the disk it reserved) for good.
	downloadTimeout = 5 * time.Minute
)

// errTestDataBusy means no disk became free within the wait. Retrying later can succeed.
var errTestDataBusy = errors.New("no disk available for the test case archive")

// TestDataConfig sizes the test data cache.
type TestDataConfig struct {
	// Dir holds the cached archives. Its contents are wiped at startup.
	Dir string
	// BudgetBytes is the most disk the cached archives may take. It must cover
	// maxConcurrent archives of the largest allowed size, or a judging could wait on one
	// that never frees: the worker checks that at startup.
	BudgetBytes int64
	Limits      TestDataLimits
	// ReserveWait defaults to defaultReserveWait.
	ReserveWait time.Duration
}

type cacheEntry struct {
	key      string
	path     string
	size     int64
	refs     int
	lastUsed time.Time
	ready    chan struct{}
	err      error
}

// testDataCache keeps test case archives on disk, so a judging reads its cases from a
// local file and never holds a problem's data in memory.
//
// The key is the archive's object key. An upload writes under a fresh UUID and the old
// key is deleted, so a key never changes content: there is nothing to invalidate.
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
	if cfg.BudgetBytes < cfg.Limits.MaxZipBytes {
		return nil, fmt.Errorf("test data budget %d is under the largest allowed archive (%d)", cfg.BudgetBytes, cfg.Limits.MaxZipBytes)
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

// acquire returns the archive for key with its reference count raised; the caller must
// release it. Concurrent callers for one key share a single download, and a download
// that fails fails every caller waiting on it and is never kept.
func (c *testDataCache) acquire(ctx context.Context, key string) (*cacheEntry, error) {
	deadline := time.NewTimer(c.maxWait)
	defer deadline.Stop()

	size := int64(-1)
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
			if size > c.limits.MaxZipBytes {
				return nil, fmt.Errorf("%w: archive is %d bytes, over the %d byte limit", errInvalidTestData, size, c.limits.MaxZipBytes)
			}
			continue
		}

		reserved, wait := c.reserveLocked(key, size)
		c.mu.Unlock()
		if reserved != nil {
			return c.fill(ctx, key, reserved)
		}
		select {
		case <-wait:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errTestDataBusy
		}
	}
}

// reserveLocked takes size bytes of the budget for a new entry, evicting archives nobody
// holds, least recently used first. When every archive is held it returns the channel to
// wait on instead. Caller holds c.mu.
func (c *testDataCache) reserveLocked(key string, size int64) (*cacheEntry, <-chan struct{}) {
	for c.used+size > c.budget {
		victim := c.oldestIdleLocked()
		if victim == nil {
			return nil, c.changed
		}
		c.dropLocked(victim)
	}
	sum := sha256.Sum256([]byte(key))
	e := &cacheEntry{
		key:      key,
		path:     filepath.Join(c.dir, hex.EncodeToString(sum[:])+".zip"),
		size:     size,
		refs:     1,
		lastUsed: c.now(),
		ready:    make(chan struct{}),
	}
	c.entries[key] = e
	c.used += size
	return e, nil
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

func (c *testDataCache) dropLocked(e *cacheEntry) {
	delete(c.entries, e.key)
	_ = os.Remove(e.path)
	c.used -= e.size
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

// fill downloads and verifies the archive this caller reserved disk for, then publishes
// the outcome to whoever is waiting on it.
func (c *testDataCache) fill(ctx context.Context, key string, e *cacheEntry) (*cacheEntry, error) {
	err := c.download(ctx, key, e)

	c.mu.Lock()
	if err != nil {
		e.err = err
		if c.entries[key] == e {
			delete(c.entries, key)
			c.used -= e.size
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

func (c *testDataCache) download(ctx context.Context, key string, e *cacheEntry) error {
	dctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	tmp := e.path + ".tmp"
	defer os.Remove(tmp)

	rc, err := c.source.readObject(dctx, key)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
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

	if err := verifyTestCaseZip(dctx, tmp, c.limits); err != nil {
		return err
	}
	return os.Rename(tmp, e.path)
}

// release drops one reference. An archive nobody holds stays on disk until the space is
// needed, so the next judging of the same problem finds it there.
func (c *testDataCache) release(e *cacheEntry) {
	c.mu.Lock()
	e.refs--
	e.lastUsed = c.now()
	c.notifyLocked()
	c.mu.Unlock()
}

// notifyLocked wakes every judging waiting for disk. Closing and replacing a channel
// wakes all of them and loses no wake-up, which a single-slot signal would.
func (c *testDataCache) notifyLocked() {
	close(c.changed)
	c.changed = make(chan struct{})
}
