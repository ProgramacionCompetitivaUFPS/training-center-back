-- +goose Up

ALTER TABLE group_members
  ADD COLUMN added_by    UUID        REFERENCES users(id),
  ADD COLUMN join_method VARCHAR(20) NOT NULL DEFAULT 'DIRECT_ADD',
  ADD COLUMN removed_at  TIMESTAMPTZ;

ALTER TABLE group_members
  ADD CONSTRAINT chk_group_members_join_method CHECK (
    join_method IN ('DIRECT_ADD','INVITATION','REQUEST_APPROVED','OPEN_JOIN')
  );

-- Replace UNIQUE(group_id, user_id) with partial index so soft-deleted rows
-- don't block re-adding a user.
ALTER TABLE group_members DROP CONSTRAINT group_members_group_id_user_id_key;
CREATE UNIQUE INDEX idx_group_members_active
  ON group_members (group_id, user_id) WHERE removed_at IS NULL;

-- NOTE: group_membership_audit_log table is deferred to the change-role/remove-member iteration.

-- +goose Down

DROP INDEX IF EXISTS idx_group_members_active;
ALTER TABLE group_members
  ADD CONSTRAINT group_members_group_id_user_id_key UNIQUE (group_id, user_id);
ALTER TABLE group_members
  DROP CONSTRAINT IF EXISTS chk_group_members_join_method,
  DROP COLUMN removed_at,
  DROP COLUMN join_method,
  DROP COLUMN added_by;
