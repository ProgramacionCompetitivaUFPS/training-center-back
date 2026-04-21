-- +goose Up

ALTER TABLE users
    ADD COLUMN preferences JSONB;

-- +goose Down

ALTER TABLE users
    DROP COLUMN preferences;
