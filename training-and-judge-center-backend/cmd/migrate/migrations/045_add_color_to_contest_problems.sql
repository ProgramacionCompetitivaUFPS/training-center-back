-- +goose Up

ALTER TABLE contest_problems
    ADD COLUMN color TEXT,
    ADD CONSTRAINT chk_contest_problems_color CHECK (color IS NULL OR color ~ '^#[0-9a-f]{6}$');

-- +goose Down

ALTER TABLE contest_problems
    DROP CONSTRAINT IF EXISTS chk_contest_problems_color,
    DROP COLUMN IF EXISTS color;
