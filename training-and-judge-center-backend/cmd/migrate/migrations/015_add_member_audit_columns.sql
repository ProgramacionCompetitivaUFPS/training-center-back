-- +goose Up

ALTER TABLE group_members
    ADD COLUMN IF NOT EXISTS added_by    UUID        REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS join_method VARCHAR(30) NOT NULL DEFAULT 'OPEN_JOIN';

ALTER TABLE group_members
    ADD CONSTRAINT chk_group_members_join_method
    CHECK (join_method IN ('DIRECT_ADD', 'INVITATION', 'REQUEST_APPROVED', 'OPEN_JOIN'));

-- +goose Down

ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS chk_group_members_join_method,
    DROP COLUMN IF EXISTS added_by,
    DROP COLUMN IF EXISTS join_method;
