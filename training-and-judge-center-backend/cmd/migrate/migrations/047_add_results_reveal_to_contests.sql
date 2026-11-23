-- +goose Up

ALTER TABLE contests ADD COLUMN IF NOT EXISTS results_revealed_at TIMESTAMPTZ;
ALTER TABLE contests ADD COLUMN IF NOT EXISTS auto_reveal_minutes INTEGER CHECK (auto_reveal_minutes >= 0);

-- Contests that already ended keep showing their full results.
UPDATE contests SET results_revealed_at = end_time WHERE freeze_minutes > 0 AND end_time <= now();

-- +goose Down

ALTER TABLE contests DROP COLUMN IF EXISTS auto_reveal_minutes;
ALTER TABLE contests DROP COLUMN IF EXISTS results_revealed_at;
