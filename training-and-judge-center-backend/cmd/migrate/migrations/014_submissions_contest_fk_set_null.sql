-- +goose Up

-- When a contest is deleted, submissions become orphaned (contest_id = NULL)
-- rather than being deleted. This preserves submission history in the user's feed.
ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_contest_id_fkey,
    ADD CONSTRAINT submissions_contest_id_fkey
        FOREIGN KEY (contest_id) REFERENCES contests(id) ON DELETE SET NULL;

-- +goose Down

-- Nullify orphaned contest_id values before re-adding the strict FK,
-- otherwise the ADD CONSTRAINT fails if any contests were deleted after the UP migration.
UPDATE submissions SET contest_id = NULL
WHERE contest_id IS NOT NULL
  AND contest_id NOT IN (SELECT id FROM contests);

ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_contest_id_fkey,
    ADD CONSTRAINT submissions_contest_id_fkey
        FOREIGN KEY (contest_id) REFERENCES contests(id);
