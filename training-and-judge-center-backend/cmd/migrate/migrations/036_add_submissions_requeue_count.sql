-- +goose Up

-- cuántas veces el worker volvió a publicar una submission PENDING sin avance;
-- acota el recuperador de PENDING obsoletas para que nunca reintente sin fin
ALTER TABLE submissions
    ADD COLUMN requeue_count INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_submissions_stale_pending
    ON submissions (updated_at)
    WHERE status = 'PENDING';

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_stale_pending;
ALTER TABLE submissions
    DROP COLUMN IF EXISTS requeue_count;
