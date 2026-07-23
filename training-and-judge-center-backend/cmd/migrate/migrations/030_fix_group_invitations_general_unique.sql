-- +goose Up
-- The original partial unique index allowed multiple simultaneous PENDING
-- general (invitee_id IS NULL) invitations per group, since Postgres treats
-- NULLs as mutually distinct for uniqueness purposes. Folding NULL into a
-- fixed sentinel via COALESCE makes all general invitations for a group
-- collide against each other, so concurrent GenerateInvite calls can no
-- longer both insert a general invitation for the same group.
DROP INDEX IF EXISTS idx_group_invitations_pending_unique;

CREATE UNIQUE INDEX idx_group_invitations_pending_unique
    ON group_invitations (group_id, COALESCE(invitee_id::text, 'general'))
    WHERE status = 'PENDING';

-- +goose Down
DROP INDEX IF EXISTS idx_group_invitations_pending_unique;

CREATE UNIQUE INDEX idx_group_invitations_pending_unique
    ON group_invitations (group_id, invitee_id) WHERE status = 'PENDING';
