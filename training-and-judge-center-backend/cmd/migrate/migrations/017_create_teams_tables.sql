-- +goose Up
CREATE TABLE IF NOT EXISTS teams (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS teams_name_lower_idx ON teams (LOWER(name));

CREATE TABLE IF NOT EXISTS team_members (
    id        UUID PRIMARY KEY,
    team_id   UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT team_members_team_user_key UNIQUE (team_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
