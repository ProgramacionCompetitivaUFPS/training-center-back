package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
)

// A missing mount is the failure this guards: Docker would create the bind
// source as an empty directory and every checker would read nothing, so the
// worker has to refuse to start rather than judge against emptiness.
func TestEnsureSharedVolume_RejectsWhatCannotBeTheVolume(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"the volume is not mounted", filepath.Join(dir, "absent")},
		{"something that is not a directory", file},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ensureSharedVolume(tt.path); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

// 0711 is what makes a judging directory a capability: a sandbox can enter the
// path it was handed and cannot list the root to find anybody else's.
func TestEnsureSharedVolume_LeavesTheRootUnlistable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}

	if err := ensureSharedVolume(dir); err != nil {
		t.Fatalf("ensureSharedVolume: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o711 {
		t.Errorf("mode: got %#o, want %#o", got, 0o711)
	}
}

// The pool honours whatever it is handed, so the light pool losing its read-only
// mount leaves every other test green while a checker gains the ability to
// rewrite another judging's output.
func TestPoolConfigFor_OnlyTheLightPoolMountsReadOnly(t *testing.T) {
	cfg := validConfig()

	heavy := poolConfigFor(cfg, poolHeavy, time.Minute, "/judging")
	light := poolConfigFor(cfg, poolLight, time.Minute, "/judging")

	if heavy.SharedVolumeReadOnly {
		t.Error("the heavy pool mounts read-only, but it has to write the contestant's output")
	}
	if !light.SharedVolumeReadOnly {
		t.Error("the light pool mounts writable, but it only ever reads")
	}
	for _, p := range []struct {
		name string
		cfg  judgepool.PoolConfig
	}{{"heavy", heavy}, {"light", light}} {
		if p.cfg.SharedVolumeSource != "/judging" {
			t.Errorf("%s pool source: got %q, want %q", p.name, p.cfg.SharedVolumeSource, "/judging")
		}
	}
}

// The pool mounts the volume at this path and the worker reads it back through
// its own mount, so the two views have to be the same path.
func TestSharedVolumePath_IsWhatTheManifestsMount(t *testing.T) {
	if judgepool.SharedVolumePath != "/judging" {
		t.Errorf("SharedVolumePath: got %q, want %q — worker.yaml and docker-compose.yml mount that path",
			judgepool.SharedVolumePath, "/judging")
	}
}
