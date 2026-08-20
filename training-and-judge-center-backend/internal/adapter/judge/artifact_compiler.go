package judge

import (
	"bytes"
	"context"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/strutil"
)

var _ appjudge.ArtifactCompiler = (*ArtifactCompiler)(nil)

const (
	maxArtifactBytes = 64 << 20
	cleanupTimeout   = 10 * time.Second
)

type ArtifactCompiler struct {
	pool   *pool.Pool
	docker dockerExecClient
	cfg    ArtifactConfig
}

func NewArtifactCompiler(p *pool.Pool, docker dockerExecClient, cfg ArtifactConfig) *ArtifactCompiler {
	return &ArtifactCompiler{pool: p, docker: docker, cfg: cfg}
}

// Compile claims a container of the artifact's own language: images carry a
// single toolchain, so a C++ checker cannot be built in the container that runs
// a Java solution.
func (c *ArtifactCompiler) Compile(ctx context.Context, req appjudge.CompileArtifactRequest) (appjudge.CompileArtifactResult, error) {
	name := req.Role.String()
	if name == "" {
		slog.ErrorContext(ctx, "artifact_compiler: request carries no role")
		return appjudge.CompileArtifactResult{}, apperror.NewInternal()
	}
	language := req.Language.String()
	langCfg, ok := c.cfg.Languages[language]
	if !ok {
		slog.ErrorContext(ctx, "artifact_compiler: unknown language in config", "language", language)
		return appjudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()

	container, err := c.pool.Claim(compileCtx, language)
	if err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: claim container failed", "language", language, "error", err)
		return appjudge.CompileArtifactResult{}, apperror.NewInternal()
	}
	defer c.release(ctx, container)

	sourcePath := withArtifactName(langCfg.SourcePath, name)
	if _, err := c.docker.CopyToContainer(compileCtx, container.ID(), client.CopyToContainerOptions{
		DestinationPath: path.Dir(sourcePath),
		Content:         buildTar(path.Base(sourcePath), req.SourceCode, modeSource),
	}); err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: copy source failed", "container_id", container.ID(), "error", err)
		return appjudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	log, exitCode, err := c.run(compileCtx, container.ID(), withArtifactName(langCfg.CompileCmd, name))
	if err != nil {
		return appjudge.CompileArtifactResult{}, err
	}
	// A non-zero exit is the problem setter's code failing to build, not a
	// failure of ours: it travels back as a result, not as an error.
	if exitCode != 0 {
		return appjudge.CompileArtifactResult{Success: false, Log: log}, nil
	}

	artifact, err := c.extract(compileCtx, container.ID(), withArtifactName(langCfg.ArtifactPath, name))
	if err != nil {
		return appjudge.CompileArtifactResult{}, err
	}
	return appjudge.CompileArtifactResult{Success: true, Log: log, Artifact: artifact}, nil
}

// run executes cmd through sh -c, which is what lets a language chain steps
// with && in its configured command.
func (c *ArtifactCompiler) run(ctx context.Context, containerID, cmd string) (string, int, error) {
	execRes, err := c.docker.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: exec create failed", "container_id", containerID, "error", err)
		return "", 0, apperror.NewInternal()
	}

	att, err := c.docker.ExecAttach(ctx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: exec attach failed", "container_id", containerID, "error", err)
		return "", 0, apperror.NewInternal()
	}
	defer att.Conn.Close()

	var outBuf, errBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&outBuf, &errBuf, att.Reader)

	inspectRes, err := c.docker.ExecInspect(ctx, execRes.ID, client.ExecInspectOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: exec inspect failed", "container_id", containerID, "error", err)
		return "", 0, apperror.NewInternal()
	}
	return strutil.Truncate(outBuf.String()+errBuf.String(), maxCompileLogBytes), inspectRes.ExitCode, nil
}

func (c *ArtifactCompiler) extract(ctx context.Context, containerID, artifactPath string) ([]byte, error) {
	res, err := c.docker.CopyFromContainer(ctx, containerID, client.CopyFromContainerOptions{
		SourcePath: artifactPath,
	})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: copy artifact failed", "container_id", containerID, "path", artifactPath, "error", err)
		return nil, apperror.NewInternal()
	}
	defer res.Content.Close()

	// An empty artifact after a successful command means the command did not
	// produce what the config says it does; storing it would fail much later.
	artifact := extractFirstFile(res.Content, maxArtifactBytes)
	if len(artifact) == 0 {
		slog.ErrorContext(ctx, "artifact_compiler: compiled artifact is empty", "container_id", containerID, "path", artifactPath)
		return nil, apperror.NewInternal()
	}
	return artifact, nil
}

// release wipes the sandbox before the container returns to the pool, so that
// contestant code claiming it next cannot read the checker's source. Cleanup
// runs on its own deadline: a compile that timed out still has to be cleaned,
// and a container that cannot be is destroyed instead of handed back.
func (c *ArtifactCompiler) release(ctx context.Context, container *pool.Container) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	if err := runAndWait(cleanupCtx, c.docker, container.ID(), []string{"sh", "-c", "rm -rf /sandbox/*"}); err != nil {
		slog.ErrorContext(ctx, "artifact_compiler: sandbox cleanup failed, discarding container", "container_id", container.ID(), "error", err)
		c.pool.Discard(cleanupCtx, container)
		return
	}
	c.pool.Release(container)
}

func withArtifactName(cmd, name string) string {
	return strings.ReplaceAll(cmd, ArtifactNamePlaceholder, name)
}
