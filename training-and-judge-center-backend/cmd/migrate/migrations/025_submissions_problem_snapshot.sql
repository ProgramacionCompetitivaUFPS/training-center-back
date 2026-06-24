-- +goose Up

-- Allow problem_id to be NULL so submissions survive problem deletion
ALTER TABLE submissions ALTER COLUMN problem_id DROP NOT NULL;

ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_problem_id_fkey,
    ADD CONSTRAINT submissions_problem_id_fkey
        FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE SET NULL;

-- Snapshot columns so problem title and slug are preserved after deletion
ALTER TABLE submissions
    ADD COLUMN IF NOT EXISTS problem_title TEXT,
    ADD COLUMN IF NOT EXISTS problem_slug  TEXT;

UPDATE submissions s
    SET problem_title = p.title,
        problem_slug  = p.slug
    FROM problems p
    WHERE s.problem_id = p.id;

-- +goose Down

-- WARNING: this permanently deletes every submission whose problem was deleted
-- after the UP migration ran (problem_id set to NULL by ON DELETE SET NULL).
-- There is no recovery path. Do not run this in production unless those rows
-- are acceptable to lose.
DELETE FROM submissions WHERE problem_id IS NULL;

ALTER TABLE submissions ALTER COLUMN problem_id SET NOT NULL;

ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_problem_id_fkey,
    ADD CONSTRAINT submissions_problem_id_fkey
        FOREIGN KEY (problem_id) REFERENCES problems(id);

ALTER TABLE submissions
    DROP COLUMN IF EXISTS problem_title,
    DROP COLUMN IF EXISTS problem_slug;
