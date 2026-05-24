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

func (e *Executor) BeginSession(ctx context.Context, lang submission.Language) (appjudge.ExecutionSession, error) {
	lc, ok := e.cfg.Languages[lang.String()]
	if !ok {
		slog.ErrorContext(ctx, "executor: unknown language in config", "language", lang.String())
		return nil, apperror.NewInternal()
	}

	c, err := e.pool.Claim(ctx, lang.String())
	if err != nil {
		return nil, apperror.NewInternal()
	}

	return &Session{
		container: c,
		pool:      e.pool,
		docker:    e.docker,
		langCfg:   lc,
	}, nil
}
