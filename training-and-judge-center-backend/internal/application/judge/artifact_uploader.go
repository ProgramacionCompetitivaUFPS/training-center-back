package judge

import "context"

// ArtifactUploader writes a compiled checker/validator artifact to storage,
// so it can be found later by a separate judge run (e.g. ProblemProvider
// reading CheckerPath for a real submission) — unlike ValidatorRunner, which
// only needs the artifact in memory for the run happening right now.
type ArtifactUploader interface {
	Upload(ctx context.Context, path string, content []byte) error
}
