package judge

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// The problem stores the upload prefix; the archive lives inside it.
const (
	providerKey    = "problems/abc/testcases/2b0f"
	providerObject = "problems/abc/testcases/2b0f/testcases.zip"
)

func providerWithKey(t *testing.T, src *fakeSource, key *string) *TestCaseProvider {
	t.Helper()
	return &TestCaseProvider{
		cache: newCache(t, src, 1<<20),
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error {
					*(dest[0].(**string)) = key
					return nil
				}}
			},
		},
	}
}

func TestTestCaseProvider_NullKey_IsTheEmptySet(t *testing.T) {
	provider := providerWithKey(t, newFakeSource(), nil)

	set, err := provider.Open(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer set.Close()
	if len(set.Cases()) != 0 {
		t.Errorf("expected no cases, got %d", len(set.Cases()))
	}
}

func TestTestCaseProvider_ProblemNotFound(t *testing.T) {
	provider := &TestCaseProvider{
		cache: newCache(t, newFakeSource(), 1<<20),
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
			},
		},
	}

	_, err := provider.Open(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestTestCaseProvider_DBError_ReturnsInternal(t *testing.T) {
	provider := &TestCaseProvider{
		cache: newCache(t, newFakeSource(), 1<<20),
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
			},
		},
	}

	_, err := provider.Open(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestTestCaseProvider_StorageNotFound_ReturnsNotFound(t *testing.T) {
	key := providerKey
	provider := providerWithKey(t, newFakeSource(), &key) // the object was never stored

	_, err := provider.Open(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestTestCaseProvider_StorageError_ReturnsInternal(t *testing.T) {
	src := newFakeSource()
	src.objects[providerObject] = archive(t, "a")
	src.readErr = errors.New("network error")
	key := providerKey
	provider := providerWithKey(t, src, &key)

	_, err := provider.Open(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestTestCaseProvider_InvalidArchive_ReturnsInternal(t *testing.T) {
	src := newFakeSource()
	src.objects[providerObject] = []byte("not a zip at all, but long enough to be an object")
	key := providerKey
	provider := providerWithKey(t, src, &key)

	_, err := provider.Open(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestTestCaseProvider_ServesTheCasesOfTheArchiveAtTheStoredKey(t *testing.T) {
	src := newFakeSource()
	src.objects[providerObject] = zipBytes(t, map[string]string{
		"data/secret/001.in":  "5",
		"data/secret/001.ans": "25",
	}, false)
	key := providerKey
	provider := providerWithKey(t, src, &key)

	set, err := provider.Open(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer set.Close()

	if len(set.Cases()) != 1 {
		t.Fatalf("expected 1 case, got %d", len(set.Cases()))
	}
	in, err := set.OpenInput("secret/001")
	if got := readAll(t, in, err); got != "5" {
		t.Errorf("input = %q", got)
	}
	ans, err := set.OpenAnswer("secret/001")
	if got := readAll(t, ans, err); got != "25" {
		t.Errorf("answer = %q", got)
	}
}

func TestTestCaseProvider_OpenAtReadsTheGivenKeyWithoutTouchingTheDatabase(t *testing.T) {
	src := newFakeSource()
	src.objects[providerObject] = zipBytes(t, map[string]string{
		"data/secret/001.in": "a", "data/secret/001.ans": "A",
	}, false)
	provider := &TestCaseProvider{cache: newCache(t, src, 1<<20)} // no db at all

	set, err := provider.OpenAt(context.Background(), providerKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer set.Close()

	if len(set.Cases()) != 1 {
		t.Errorf("expected 1 case, got %d", len(set.Cases()))
	}
}

// What makes the cache pay off: the next judging of the same problem reads the file that is
// already there. Closing a set is what lets the archive be reused (and, later, evicted).
func TestTestCaseProvider_AClosedSetLeavesTheArchiveCachedAndUnheld(t *testing.T) {
	src := newFakeSource()
	src.objects[providerObject] = archive(t, "a")
	provider := &TestCaseProvider{cache: newCache(t, src, 1<<20)}

	for i := 0; i < 3; i++ {
		set, err := provider.OpenAt(context.Background(), providerKey)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		_ = set.Close()
	}

	if got := src.readCount(providerObject); got != 1 {
		t.Errorf("downloaded %d times for 3 judgings, want 1", got)
	}
	provider.cache.mu.Lock()
	defer provider.cache.mu.Unlock()
	for _, e := range provider.cache.entries {
		if e.refs != 0 {
			t.Errorf("an archive is still held (refs=%d) after every set was closed", e.refs)
		}
	}
}
