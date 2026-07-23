-- +goose Up

ALTER TABLE contests ADD COLUMN IF NOT EXISTS show_team_members BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE contests DROP COLUMN IF EXISTS show_team_members;
