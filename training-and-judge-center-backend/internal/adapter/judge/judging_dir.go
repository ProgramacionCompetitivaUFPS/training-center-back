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

// createJudgingDir lays out the slot. The worker is root, so it keeps writing
// inside a directory it just made unwritable for everyone else.
func createJudgingDir(judgingDir string) error {
	if err := os.Mkdir(judgingDir, judgingDirMode); err != nil {
		return err
	}
	out := judgingOutputPath(judgingDir)
	if err := os.WriteFile(out, nil, judgingFileMode); err != nil {
		return err
	}
	// The sandbox rewrites this file on every test case, so it has to own it.
	return os.Chown(out, sandboxUID, sandboxUID)
}
