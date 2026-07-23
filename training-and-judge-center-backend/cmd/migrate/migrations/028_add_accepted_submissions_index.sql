-- +goose Up

-- Partial index on (user_id, problem_id) for ACCEPTED submissions only.
-- Enables index-only scans for per-user unique-problem counts and global ranking
-- queries in the dashboard, replacing full-table COUNT(DISTINCT) aggregations.
CREATE INDEX idx_submissions_accepted_user_problem
    ON submissions (user_id, problem_id)
    WHERE status = 'ACCEPTED';

-- +goose Down
DROP INDEX IF EXISTS idx_submissions_accepted_user_problem;
