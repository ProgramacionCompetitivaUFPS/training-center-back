package judge

import (
	"os"
	"path"
)

// One judging's slot in the shared volume: the heavy pool writes the contestant's
// output there and the light pool reads it, so neither travels through the
// worker. The directory's name is the credential — the volume's root is not
// listable, so a sandbox reaches only the path it was handed.
const (
	// sandboxUID is the judge user, fixed at 1000 in the runner base image.
	sandboxUID = 1000
	// The sandbox can enter the directory but neither list nor write it, so
	// output.txt is pre-created for the shell redirection to reuse.
	judgingDirMode  = 0o111
	judgingFileMode = 0o644
)

func judgingInputPath(judgingDir string) string {
	return path.Join(judgingDir, "input.txt")
}

func judgingOutputPath(judgingDir string) string {
	return path.Join(judgingDir, "output.txt")
}

// judgingMemPath is where /usr/bin/time leaves the run's peak RSS.
func judgingMemPath(judgingDir string) string {
	return path.Join(judgingDir, "mem.txt")
}

// createJudgingDir lays out the slot, then locks the directory: the sandbox must
// reach the two files it is given and be unable to add, list or remove anything.
func createJudgingDir(judgingDir string) error {
	// Created writable and tightened at the end: the final mode forbids
	// creating anything, the two files below included.
	if err := os.Mkdir(judgingDir, 0o700); err != nil {
		return err
	}
	// Both are rewritten by the sandbox on every test case, so it has to own them.
	for _, p := range []string{judgingOutputPath(judgingDir), judgingMemPath(judgingDir)} {
		if err := os.WriteFile(p, nil, judgingFileMode); err != nil {
			return err
		}
		// Handing a file to another user needs root, which the worker container is.
		if os.Geteuid() == 0 {
			if err := os.Chown(p, sandboxUID, sandboxUID); err != nil {
				return err
			}
		}
	}
	return os.Chmod(judgingDir, judgingDirMode)
}

// removeJudgingDir undoes the lock before deleting: a directory in
// judgingDirMode cannot have its entries removed, not even by its owner.
func removeJudgingDir(judgingDir string) error {
	_ = os.Chmod(judgingDir, 0o700)
	return os.RemoveAll(judgingDir)
}
