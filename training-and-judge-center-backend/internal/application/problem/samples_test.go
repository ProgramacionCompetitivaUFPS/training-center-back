package problem

import (
	"context"
	"testing"
)

func TestLoadSamples_NoTestCasesUploadedYet(t *testing.T) {
	samples, err := loadSamples(context.Background(), &mockFileStorage{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("expected no samples, got %d", len(samples))
	}
}

func TestLoadSamples_PairsInputAndOutputByName(t *testing.T) {
	key := "problems/test-problem/testcases/xyz"
	storage := &mockFileStorage{
		listFilesFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{
				"problems/test-problem/testcases/xyz/data/sample/2.in",
				"problems/test-problem/testcases/xyz/data/sample/2.ans",
				"problems/test-problem/testcases/xyz/data/sample/1.in",
				"problems/test-problem/testcases/xyz/data/sample/1.ans",
			}, nil
		},
		downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
			return []byte(path), nil
		},
	}

	samples, err := loadSamples(context.Background(), storage, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[0].Name != "1" || samples[1].Name != "2" {
		t.Errorf("expected samples sorted by name [1, 2], got [%s, %s]", samples[0].Name, samples[1].Name)
	}
}

func TestLoadSamples_IgnoresUnpairedFiles(t *testing.T) {
	key := "problems/test-problem/testcases/xyz"
	storage := &mockFileStorage{
		listFilesFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{
				"problems/test-problem/testcases/xyz/data/sample/1.in",
				"problems/test-problem/testcases/xyz/data/sample/1.ans",
				"problems/test-problem/testcases/xyz/data/sample/2.in",
			}, nil
		},
		downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
			return []byte(path), nil
		},
	}

	samples, err := loadSamples(context.Background(), storage, &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 paired sample, got %d", len(samples))
	}
	if samples[0].Name != "1" {
		t.Errorf("expected sample %q, got %q", "1", samples[0].Name)
	}
}
