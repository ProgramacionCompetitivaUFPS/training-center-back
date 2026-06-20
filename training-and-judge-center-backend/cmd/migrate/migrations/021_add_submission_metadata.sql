-- +goose Up

ALTER TABLE submissions
    ADD COLUMN file_hash   TEXT,
    ADD COLUMN file_size   INTEGER,
    ADD COLUMN compiler    TEXT,
    ADD COLUMN standing_id UUID;

-- duplicate-detection index: same hash from same user to same problem
CREATE INDEX IF NOT EXISTS idx_submissions_hash_user_problem
    ON submissions (file_hash, user_id, problem_id)
    WHERE file_hash IS NOT NULL;

-- rate-limit index: most-recent submission per user+problem
CREATE INDEX IF NOT EXISTS idx_submissions_rate_limit
    ON submissions (user_id, problem_id, submitted_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_hash_user_problem;
DROP INDEX IF EXISTS idx_submissions_rate_limit;
ALTER TABLE submissions
    DROP COLUMN IF EXISTS file_hash,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS compiler,
    DROP COLUMN IF EXISTS standing_id;
