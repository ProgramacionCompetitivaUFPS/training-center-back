-- +goose Up

CREATE TABLE IF NOT EXISTS problem_validations (
    id           UUID PRIMARY KEY,
    problem_id   UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    status       TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    result       JSONB
);

-- a lo sumo una validación PENDING/RUNNING por problema — evita duplicados incluso
-- ante requests concurrentes (ver PublishProblemUseCase, Fase 4)
CREATE UNIQUE INDEX IF NOT EXISTS idx_problem_validations_active_per_problem
    ON problem_validations (problem_id)
    WHERE status IN ('PENDING', 'RUNNING');

CREATE INDEX IF NOT EXISTS idx_problem_validations_problem_id
    ON problem_validations (problem_id, requested_at DESC);

-- +goose Down

DROP TABLE IF EXISTS problem_validations;
