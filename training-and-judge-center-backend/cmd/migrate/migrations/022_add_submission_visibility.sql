-- +goose Up

ALTER TABLE submissions
    ADD COLUMN visibility TEXT NOT NULL DEFAULT 'PRIVATE';

-- +goose Down

ALTER TABLE submissions
    DROP COLUMN IF EXISTS visibility;
