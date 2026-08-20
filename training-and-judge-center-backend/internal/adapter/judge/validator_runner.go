package judge

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path"

	"cloud.google.com/go/storage"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.ValidatorRunner = (*ValidatorRunner)(nil)

type ValidatorRunner struct {
	pool   *pool.Pool
	docker dockerExecClient
	reader gcsReader
	cfg    ArtifactConfig
}

func NewValidatorRunner(p *pool.Pool, docker dockerExecClient, cfg ArtifactConfig, client *storage.Client, bucket string) *ValidatorRunner {
	return &ValidatorRunner{pool: p, docker: docker, reader: newGCSReader(client, bucket), cfg: cfg}
}

// BeginValidating downloads the compiled validator and injects it into a light
// pool container, which then runs it against every input of the problem.
func (r *ValidatorRunner) BeginValidating(ctx context.Context, validatorPath string, language submission.Language) (appjudge.ValidatorSession, error) {
	lang := language.String()
	langCfg, ok := r.cfg.Languages[lang]
	if !ok {
		slog.ErrorContext(ctx, "validator_runner: unknown language in config", "language", lang)
		return nil, apperror.NewInternal()
	}

	// Downloading first keeps the failure path simple: no container is held
	// while storage is being read.
	artifact, err := r.download(ctx, validatorPath)
	if err != nil {
		return nil, err
	}

	container, err := r.pool.Claim(ctx, lang)
	if err != nil {
		slog.ErrorContext(ctx, "validator_runner: claim container failed", "language", lang, "error", err)
		return nil, apperror.NewInternal()
	}

	name := appjudge.NewArtifactRoleValidator().String()
	s := &ValidatorSession{
		container: container,
		pool:      r.pool,
		docker:    r.docker,
		runCmd:    withArtifactName(langCfg.RunCmd, name),
	}

	artifactPath := withArtifactName(langCfg.ArtifactPath, name)
	if _, err := r.docker.CopyToContainer(ctx, container.ID(), client.CopyToContainerOptions{
		DestinationPath: path.Dir(artifactPath),
		Content:         buildTar(path.Base(artifactPath), artifact, modeExecutable),
	}); err != nil {
		slog.ErrorContext(ctx, "validator_runner: copy artifact failed", "container_id", container.ID(), "error", err)
		// The container cannot go back dirty, so it is returned the same way a
		// finished session returns it.
		_ = s.Close(ctx)
		return nil, apperror.NewInternal()
	}
	return s, nil
}

func (r *ValidatorRunner) download(ctx context.Context, validatorPath string) ([]byte, error) {
	rc, err := r.reader.readObject(ctx, validatorPath)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "validator_runner: compiled validator not found", "path", validatorPath)
		} else {
			slog.ErrorContext(ctx, "validator_runner: failed to open the compiled validator", "path", validatorPath, "error", err)
		}
		return nil, apperror.NewInternal()
	}
	defer rc.Close()

	artifact, err := io.ReadAll(&io.LimitedReader{R: rc, N: maxArtifactBytes})
	if err != nil {
		slog.ErrorContext(ctx, "validator_runner: failed to read the compiled validator", "path", validatorPath, "error", err)
		return nil, apperror.NewInternal()
	}
	if len(artifact) == 0 {
		slog.ErrorContext(ctx, "validator_runner: the compiled validator is empty", "path", validatorPath)
		return nil, apperror.NewInternal()
	}
	return artifact, nil
}
