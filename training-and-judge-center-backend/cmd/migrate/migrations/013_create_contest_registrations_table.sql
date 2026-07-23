-- +goose Up

CREATE TABLE IF NOT EXISTS contest_registrations (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id    UUID        NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (contest_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_contest_registrations_contest_id ON contest_registrations (contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_registrations_user_id    ON contest_registrations (user_id);

-- +goose Down

DROP TABLE IF EXISTS contest_registrations;
