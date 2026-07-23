-- +goose Up

ALTER TABLE contests
    ADD COLUMN participation_mode VARCHAR(20)  NOT NULL DEFAULT 'INDIVIDUAL',
    ADD COLUMN team_size_min      INTEGER       NOT NULL DEFAULT 2,
    ADD COLUMN team_size_max      INTEGER       NOT NULL DEFAULT 5;

ALTER TABLE contests
    ADD CONSTRAINT chk_contests_participation_mode
        CHECK (participation_mode IN ('INDIVIDUAL', 'TEAM', 'MIXED')),
    ADD CONSTRAINT chk_contests_team_size_min
        CHECK (team_size_min >= 1),
    ADD CONSTRAINT chk_contests_team_size_range
        CHECK (team_size_max >= team_size_min);

-- +goose Down

ALTER TABLE contests
    DROP CONSTRAINT IF EXISTS chk_contests_team_size_range,
    DROP CONSTRAINT IF EXISTS chk_contests_team_size_min,
    DROP CONSTRAINT IF EXISTS chk_contests_participation_mode,
    DROP COLUMN IF EXISTS team_size_max,
    DROP COLUMN IF EXISTS team_size_min,
    DROP COLUMN IF EXISTS participation_mode;
