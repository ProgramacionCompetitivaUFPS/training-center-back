-- +goose Up
ALTER TABLE users ALTER COLUMN password DROP NOT NULL;

CREATE TABLE IF NOT EXISTS oauth_identities (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider          VARCHAR(20) NOT NULL,
    provider_user_id  VARCHAR(255) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_identities_provider_user
    ON oauth_identities (provider, provider_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_identities_user_provider
    ON oauth_identities (user_id, provider);

-- +goose Down
DROP TABLE IF EXISTS oauth_identities;
ALTER TABLE users ALTER COLUMN password SET NOT NULL;
