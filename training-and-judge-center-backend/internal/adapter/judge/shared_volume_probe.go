package judge

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
)

// probeMarker is what the worker writes and the sandbox has to read back.
const probeMarker = "shared-volume-probe"

// VerifySharedVolume writes a file into the volume and has a sandbox read it
// back. Nothing else can catch a mount whose source the daemon resolves
// elsewhere: Docker turns a source it cannot find into an empty directory
// rather than an error, so the judging would run and every checker would
// compare against nothing.
func VerifySharedVolume(ctx context.Context, lightPool *pool.Pool, docker dockerExecClient, judgingRoot string) error {
	dir := path.Join(judgingRoot, "probe")
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clearing the probe directory: %w", err)
	}
	if err := createJudgingDir(dir); err != nil {
		return fmt.Errorf("laying out the probe directory: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(judgingOutputPath(dir), []byte(probeMarker), judgingFileMode); err != nil {
		return fmt.Errorf("writing the probe marker: %w", err)
	}

	container, err := lightPool.Claim(ctx, CompareLanguage, pool.LanguageCeiling)
	if err != nil {
		return fmt.Errorf("claiming a %s container: %w", CompareLanguage, err)
	}
	defer lightPool.Release(container)

	got, err := readFileInContainer(ctx, docker, container.ID(), judgingOutputPath(dir))
	if err != nil {
		return fmt.Errorf("reading the marker from a sandbox: %w", err)
	}
	if got != probeMarker {
		return fmt.Errorf("a sandbox read %q where the worker wrote %q: the volume is not the same one on both sides", got, probeMarker)
	}
	return nil
}
