-- +goose Up

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS preferences JSONB;

-- +goose Down

ALTER TABLE users
    DROP COLUMN preferences;
