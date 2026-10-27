-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id                  UUID PRIMARY KEY,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id           UUID NOT NULL,
    token_hash          BYTEA NOT NULL,
    issued_at           TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    replaced_by_id      UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent          TEXT,
    ip_address          INET,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_active ON refresh_tokens (family_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_active ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
