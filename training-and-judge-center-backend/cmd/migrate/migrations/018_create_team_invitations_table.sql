-- +goose Up
CREATE TABLE IF NOT EXISTS team_invitations (
    id           UUID PRIMARY KEY,
    team_id      UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    invitee_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invited_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL,
    CONSTRAINT team_invitations_team_invitee_key UNIQUE (team_id, invitee_id)
);

-- +goose Down
DROP TABLE IF EXISTS team_invitations;
