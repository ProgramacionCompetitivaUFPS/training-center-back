-- +goose Up

CREATE TABLE IF NOT EXISTS contest_team_participants (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id       UUID        NOT NULL REFERENCES contests(id)  ON DELETE CASCADE,
    team_id          UUID        NOT NULL REFERENCES teams(id)     ON DELETE CASCADE,
    selected_members UUID[]      NOT NULL DEFAULT '{}',
    registered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_contest_team_participants_pair UNIQUE (contest_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_ctp_contest_id ON contest_team_participants (contest_id);
CREATE INDEX IF NOT EXISTS idx_ctp_team_id    ON contest_team_participants (team_id);

-- +goose Down

DROP TABLE IF EXISTS contest_team_participants;
