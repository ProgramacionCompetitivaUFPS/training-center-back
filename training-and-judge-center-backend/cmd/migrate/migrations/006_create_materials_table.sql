-- +goose Up
CREATE TABLE IF NOT EXISTS materials (
    id           UUID PRIMARY KEY,
    title        VARCHAR(200) NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    tags         TEXT[] NOT NULL DEFAULT '{}',
    status       VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    pinned       BOOLEAN NOT NULL DEFAULT FALSE,
    pinned_at    TIMESTAMPTZ,
    group_id     UUID NOT NULL,
    author_id    UUID NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', title), 'A') ||
        setweight(to_tsvector('english', coalesce(content, '')), 'B')
    ) STORED
);

CREATE INDEX idx_materials_group_id ON materials(group_id);
CREATE INDEX idx_materials_author_id ON materials(author_id);
CREATE INDEX idx_materials_group_status ON materials(group_id, status, published_at DESC NULLS LAST);
CREATE INDEX idx_materials_pinned ON materials(group_id, pinned, pinned_at DESC NULLS LAST);
CREATE INDEX idx_materials_search ON materials USING GIN(search_vector);

-- +goose Down
DROP TABLE IF EXISTS materials;
