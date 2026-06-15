-- +goose Up

-- When a contest is deleted, submissions become orphaned (contest_id = NULL)
-- rather than being deleted. This preserves submission history in the user's feed.
ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_contest_id_fkey,
    ADD CONSTRAINT submissions_contest_id_fkey
        FOREIGN KEY (contest_id) REFERENCES contests(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_contest_id_fkey,
    ADD CONSTRAINT submissions_contest_id_fkey
        FOREIGN KEY (contest_id) REFERENCES contests(id);
