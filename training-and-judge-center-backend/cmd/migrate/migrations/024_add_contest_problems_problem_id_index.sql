-- +goose Up
CREATE INDEX idx_contest_problems_problem_id ON contest_problems (problem_id);

-- +goose Down
DROP INDEX IF EXISTS idx_contest_problems_problem_id;
