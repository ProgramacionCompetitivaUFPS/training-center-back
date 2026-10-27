-- +goose Up

-- queued_at: cuándo el broker confirmó el mensaje de la submission. PENDING con
-- queued_at NULL es un mensaje que nunca llegó a la cola (determinista, no un
-- umbral de espera): solo esas las recupera el worker.
-- requeue_count acota cuántas veces se recupera una misma submission.
ALTER TABLE submissions
    ADD COLUMN queued_at     TIMESTAMPTZ,
    ADD COLUMN requeue_count INT NOT NULL DEFAULT 0;

-- lo que ya existe se da por encolado: no hay forma de saber lo contrario
UPDATE submissions SET queued_at = submitted_at;

CREATE INDEX IF NOT EXISTS idx_submissions_unqueued_pending
    ON submissions (updated_at)
    WHERE status = 'PENDING' AND queued_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_unqueued_pending;
ALTER TABLE submissions
    DROP COLUMN IF EXISTS requeue_count,
    DROP COLUMN IF EXISTS queued_at;
