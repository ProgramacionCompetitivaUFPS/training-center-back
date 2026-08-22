package judge

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Executor struct {
	pool   *pool.Pool
	docker dockerExecClient
	cfg    ExecutorConfig
}

func NewExecutor(p *pool.Pool, docker dockerExecClient, cfg ExecutorConfig) *Executor {
	return &Executor{pool: p, docker: docker, cfg: cfg}
}

func (e *Executor) BeginSession(ctx context.Context, lang submission.Language, memoryKb int) (appjudge.ExecutionSession, error) {
	lc, ok := e.cfg.Languages[lang.String()]
	if !ok {
		slog.ErrorContext(ctx, "executor: unknown language in config", "language", lang.String())
		return nil, apperror.NewInternal()
	}

	// The container enforces the problem's limit, so MLE fires where the problem
	// says and not at the pool's ceiling.
	c, err := e.pool.Claim(ctx, lang.String(), int64(memoryKb)*1024)
	if err != nil {
		slog.ErrorContext(ctx, "executor: claim container failed", "language", lang.String(), "error", err)
		return nil, apperror.NewInternal()
	}

	return &Session{
		container: c,
		pool:      e.pool,
		docker:    e.docker,
		langCfg:   lc,
	}, nil
}
