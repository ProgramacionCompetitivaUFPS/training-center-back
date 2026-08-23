package judge

import (
	"context"
	"log/slog"
	"path"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Executor struct {
	pool   *pool.Pool
	docker dockerExecClient
	cfg    ExecutorConfig
	// judgingRoot is the shared volume, injected so tests can use a real
	// directory instead of the one the manifests mount.
	judgingRoot string
}

func NewExecutor(p *pool.Pool, docker dockerExecClient, cfg ExecutorConfig, judgingRoot string) *Executor {
	return &Executor{pool: p, docker: docker, cfg: cfg, judgingRoot: judgingRoot}
}

func (e *Executor) BeginSession(ctx context.Context, lang submission.Language, memoryKb int, judgingID string) (appjudge.ExecutionSession, error) {
	lc, ok := e.cfg.Languages[lang.String()]
	if !ok {
		slog.ErrorContext(ctx, "executor: unknown language in config", "language", lang.String())
		return nil, apperror.NewInternal()
	}

	// Laid out before claiming, so a failure here leaves no container held.
	judgingDir := path.Join(e.judgingRoot, judgingID)
	if err := createJudgingDir(judgingDir); err != nil {
		slog.ErrorContext(ctx, "executor: judging directory setup failed", "error", err)
		return nil, apperror.NewInternal()
	}

	// The container enforces the problem's limit, so MLE fires where the problem
	// says and not at the pool's ceiling. The factor buys back what the runtime
	// reserves for itself, so the same limit means the same usable memory in
	// every language.
	c, err := e.pool.Claim(ctx, lang.String(), int64(float64(memoryKb)*1024*lc.MemoryFactor))
	if err != nil {
		_ = removeJudgingDir(judgingDir)
		slog.ErrorContext(ctx, "executor: claim container failed", "language", lang.String(), "error", err)
		return nil, apperror.NewInternal()
	}

	return &Session{
		container:  c,
		pool:       e.pool,
		docker:     e.docker,
		langCfg:    lc,
		judgingDir: judgingDir,
	}, nil
}
