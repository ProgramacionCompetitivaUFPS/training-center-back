package judge

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The layout is what keeps one judging out of another: the sandbox can enter a
// directory whose path it was handed, but cannot list the root to find anyone
// else's, nor create or delete anything inside its own.
func TestCreateJudgingDir_LeavesTheSlotUnlistableAndUnwritable(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("needs root to chown output.txt to the sandbox user")
	}
	dir := filepath.Join(t.TempDir(), "judging-1")

	if err := createJudgingDir(dir); err != nil {
		t.Fatalf("createJudgingDir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	// The literal, not judgingDirMode: comparing the constant against itself
	// would pass no matter what the constant said.
	if got := info.Mode().Perm(); got != 0o111 {
		t.Errorf("directory mode: got %#o, want %#o — the sandbox must not list or write it", got, 0o111)
	}

	// Both are pre-created because the mode above forbids the sandbox creating
	// anything: the shell redirection and time -o would fail otherwise.
	for _, p := range []string{judgingOutputPath(dir), judgingMemPath(dir)} {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("%s was not pre-created: %v", filepath.Base(p), err)
			continue
		}
		if uid := sandboxOwner(t, info); uid != sandboxUID {
			t.Errorf("%s owner: got uid %d, want the sandbox user %d", filepath.Base(p), uid, sandboxUID)
		}
	}
}

// Twice over the same name is a name collision, not a directory to reuse: two
// judgings sharing a slot would read each other's files.
func TestCreateJudgingDir_RefusesAnExistingSlot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "judging-1")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := createJudgingDir(dir); err == nil {
		t.Error("expected an error for a slot that already exists, got nil")
	}
}

// sandboxOwner reads the uid off a FileInfo, which needs the syscall-level stat
// Go keeps behind Sys().
func sandboxOwner(t *testing.T, info os.FileInfo) int {
	t.Helper()
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("no unix stat behind %T", info.Sys())
	}
	return int(stat.Uid)
}
