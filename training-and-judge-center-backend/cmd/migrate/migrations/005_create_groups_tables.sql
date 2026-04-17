-- +goose Up

CREATE TABLE groups (
    id           UUID         PRIMARY KEY,
    name         VARCHAR(300) NOT NULL,
    description  TEXT,
    visibility   VARCHAR(20)  NOT NULL,
    join_policy  VARCHAR(20)  NOT NULL,
    is_default   BOOLEAN      NOT NULL DEFAULT false,
    created_by   UUID         NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_groups_visibility  CHECK (visibility  IN ('VISIBLE', 'NOT_VISIBLE')),
    CONSTRAINT chk_groups_join_policy CHECK (join_policy IN ('INVITE', 'REQUEST', 'OPEN'))
);

CREATE UNIQUE INDEX idx_groups_name ON groups (LOWER(name));

CREATE TABLE group_members (
    id          UUID        PRIMARY KEY,
    group_id    UUID        NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id),
    member_role VARCHAR(20) NOT NULL DEFAULT 'MEMBER',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, user_id),
    CONSTRAINT chk_group_members_role CHECK (member_role IN ('LEAD', 'MEMBER'))
);

CREATE INDEX idx_group_members_group_id ON group_members (group_id);
CREATE INDEX idx_group_members_user_id  ON group_members (user_id);

-- +goose Down

DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
