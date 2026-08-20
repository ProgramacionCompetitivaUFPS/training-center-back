package judge

import (
	"context"
	"log/slog"
	"path"

	"cloud.google.com/go/storage"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.OutputChecker = (*OutputChecker)(nil)

type OutputChecker struct {
	pool   *pool.Pool
	docker dockerExecClient
	reader gcsReader
	cfg    ArtifactConfig
}

func NewOutputChecker(p *pool.Pool, docker dockerExecClient, cfg ArtifactConfig, client *storage.Client, bucket string) *OutputChecker {
	return &OutputChecker{pool: p, docker: docker, reader: newGCSReader(client, bucket), cfg: cfg}
}

// BeginChecking downloads the compiled checker and injects it into a light pool
// container, which then checks every output of the judging.
func (c *OutputChecker) BeginChecking(ctx context.Context, checkerPath string, language submission.Language) (appjudge.CheckerSession, error) {
	// Token comparison still runs in the worker, so no container is claimed for a
	// problem without a custom checker. Step 7 moves it to the light pool.
	if checkerPath == "" {
		return &tokenCheckerSession{}, nil
	}

	lang := language.String()
	langCfg, ok := c.cfg.Languages[lang]
	if !ok {
		slog.ErrorContext(ctx, "output_checker: unknown language in config", "language", lang)
		return nil, apperror.NewInternal()
	}

	// Downloading first keeps the failure path simple: no container is held
	// while storage is being read.
	artifact, err := downloadArtifact(ctx, c.reader, checkerPath)
	if err != nil {
		return nil, err
	}

	container, err := c.pool.Claim(ctx, lang)
	if err != nil {
		slog.ErrorContext(ctx, "output_checker: claim container failed", "language", lang, "error", err)
		return nil, apperror.NewInternal()
	}

	name := appjudge.NewArtifactRoleChecker().String()
	s := &CheckerSession{artifactSession{
		container: container,
		pool:      c.pool,
		docker:    c.docker,
		role:      name,
		runCmd:    withArtifactName(langCfg.RunCmd, name),
	}}

	artifactPath := withArtifactName(langCfg.ArtifactPath, name)
	if _, err := c.docker.CopyToContainer(ctx, container.ID(), client.CopyToContainerOptions{
		DestinationPath: path.Dir(artifactPath),
		Content:         buildTar(path.Base(artifactPath), artifact, modeExecutable),
	}); err != nil {
		slog.ErrorContext(ctx, "output_checker: copy artifact failed", "container_id", container.ID(), "error", err)
		// The container cannot go back dirty, so it is returned the same way a
		// finished session returns it.
		_ = s.Close(ctx)
		return nil, apperror.NewInternal()
	}
	return s, nil
}
