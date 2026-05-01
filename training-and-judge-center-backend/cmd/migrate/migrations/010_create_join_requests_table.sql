-- +goose Up
CREATE TABLE join_requests (
    id                UUID        PRIMARY KEY,
    group_id          UUID        NOT NULL REFERENCES groups(id),
    requester_user_id UUID        NOT NULL REFERENCES users(id),
    message           TEXT,
    status            VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at        TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX idx_join_requests_pending
    ON join_requests(group_id, requester_user_id)
    WHERE status = 'PENDING';

CREATE INDEX idx_join_requests_group_status
    ON join_requests(group_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_join_requests_group_status;
DROP INDEX IF EXISTS idx_join_requests_pending;
DROP TABLE IF EXISTS join_requests;
