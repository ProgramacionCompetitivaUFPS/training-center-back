-- +goose Up
CREATE TABLE IF NOT EXISTS group_invitations (
    id          UUID PRIMARY KEY,
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    invitee_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    invited_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

-- Partial unique index: only one PENDING invitation per (group, invitee) pair.
-- NULLs are distinct from each other for uniqueness purposes in Postgres, so
-- this does not by itself prevent multiple concurrent PENDING general
-- invitations (invitee_id IS NULL) for the same group — GenerateInviteUseCase
-- avoids that in practice by revoking the existing general invitation before
-- inserting a new one, inside the same transaction.
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_invitations_pending_unique
    ON group_invitations (group_id, invitee_id) WHERE status = 'PENDING';

-- Matches the WHERE group_id = $1 AND status = $2 query shape used by
-- ListGroupInvitationsUseCase, same criterion as idx_join_requests_group_status.
CREATE INDEX IF NOT EXISTS idx_group_invitations_group_status ON group_invitations (group_id, status);

-- +goose Down
DROP TABLE IF EXISTS group_invitations;
