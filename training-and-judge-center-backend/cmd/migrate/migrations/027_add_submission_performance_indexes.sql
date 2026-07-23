-- +goose Up

-- Standings and rejudge: filter by contest + terminal status
CREATE INDEX idx_submissions_contest_id_status
    ON submissions (contest_id, status)
    WHERE contest_id IS NOT NULL
      AND status NOT IN ('PENDING', 'RUNNING');

-- Accepted-only standings score (rank by last accepted)
CREATE INDEX idx_submissions_contest_id_status_user_id
    ON submissions (contest_id, status, user_id)
    WHERE contest_id IS NOT NULL
      AND status = 'ACCEPTED';

-- Problem statistics: terminal submissions per problem ordered by time
CREATE INDEX idx_submissions_problem_id_submitted_at
    ON submissions (problem_id, submitted_at)
    WHERE status NOT IN ('PENDING', 'RUNNING');

-- Dashboard streak: per-user submission history ordered by time
CREATE INDEX idx_submissions_user_id_submitted_at
    ON submissions (user_id, submitted_at DESC);

-- Dashboard / user listing: filter active users
CREATE INDEX idx_users_status_active
    ON users (status)
    WHERE status = 'ACTIVE';

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_contest_id_status;
DROP INDEX IF EXISTS idx_submissions_contest_id_status_user_id;
DROP INDEX IF EXISTS idx_submissions_problem_id_submitted_at;
DROP INDEX IF EXISTS idx_submissions_user_id_submitted_at;
DROP INDEX IF EXISTS idx_users_status_active;
