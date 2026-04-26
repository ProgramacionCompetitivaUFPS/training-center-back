-- +goose Up

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS preferences JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down

ALTER TABLE users
    DROP COLUMN preferences;
