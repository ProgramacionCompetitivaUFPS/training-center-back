-- +goose Up

CREATE TABLE IF NOT EXISTS submissions (
    id               TEXT        PRIMARY KEY,
    problem_id       UUID        NOT NULL REFERENCES problems(id),
    user_id          UUID        NOT NULL REFERENCES users(id),
    contest_id       UUID        REFERENCES contests(id),
    language         TEXT        NOT NULL,
    status           TEXT        NOT NULL,
    source_code_path TEXT        NOT NULL,
    submitted_at     TIMESTAMPTZ NOT NULL,
    judged_at        TIMESTAMPTZ,
    time_ms          INTEGER,
    memory_kb        INTEGER,
    compile_log      TEXT
);

CREATE INDEX IF NOT EXISTS idx_submissions_user_id    ON submissions (user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_id ON submissions (problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_contest_id ON submissions (contest_id) WHERE contest_id IS NOT NULL;

-- +goose Down

DROP TABLE IF EXISTS submissions;
