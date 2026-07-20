-- +goose Up

ALTER TABLE submissions
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- stale-recovery index: cheap scans for submissions stuck in RUNNING
CREATE INDEX IF NOT EXISTS idx_submissions_stale_running
    ON submissions (updated_at)
    WHERE status = 'RUNNING';

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_stale_running;
ALTER TABLE submissions
    DROP COLUMN IF EXISTS updated_at;
