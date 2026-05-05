-- +goose Up

CREATE TABLE IF NOT EXISTS contests (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 VARCHAR(200)  NOT NULL,
    description          TEXT,
    start_time           TIMESTAMPTZ   NOT NULL,
    end_time             TIMESTAMPTZ   NOT NULL,
    penalty              INTEGER       NOT NULL DEFAULT 20,
    freeze_minutes       INTEGER       NOT NULL DEFAULT 60,
    enable_post_contest  BOOLEAN       NOT NULL DEFAULT FALSE,
    locked               BOOLEAN       NOT NULL DEFAULT FALSE,
    group_id             UUID          NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    owner_id             UUID          NOT NULL REFERENCES users(id),
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ,

    CONSTRAINT chk_contests_end_after_start  CHECK (end_time > start_time),
    CONSTRAINT chk_contests_penalty_range    CHECK (penalty >= 0 AND penalty <= 1440),
    CONSTRAINT chk_contests_freeze_non_neg   CHECK (freeze_minutes >= 0)
);

CREATE INDEX IF NOT EXISTS idx_contests_group_id    ON contests (group_id);
CREATE INDEX IF NOT EXISTS idx_contests_start_time  ON contests (start_time);

CREATE TABLE IF NOT EXISTS contest_problems (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id  UUID    NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id  UUID    NOT NULL REFERENCES problems(id),
    "order"     INTEGER NOT NULL,

    CONSTRAINT uq_contest_problems_pair   UNIQUE (contest_id, problem_id),
    CONSTRAINT uq_contest_problems_order  UNIQUE (contest_id, "order"),
    CONSTRAINT chk_contest_problems_order CHECK ("order" >= 1)
);

CREATE INDEX IF NOT EXISTS idx_contest_problems_contest_id ON contest_problems (contest_id);

-- +goose Down

DROP TABLE IF EXISTS contest_problems;
DROP TABLE IF EXISTS contests;
