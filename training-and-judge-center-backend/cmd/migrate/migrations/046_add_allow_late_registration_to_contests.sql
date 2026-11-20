-- +goose Up

ALTER TABLE contests ADD COLUMN IF NOT EXISTS allow_late_registration BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE contests DROP COLUMN IF EXISTS allow_late_registration;
