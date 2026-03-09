-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS problems (
    id UUID PRIMARY KEY,
    slug VARCHAR(70) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    statement TEXT,
    time_limit INTEGER,
    memory_limit INTEGER,
    tags TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    accessibility VARCHAR(50) NOT NULL DEFAULT 'PRIVATE',
    author_id UUID NOT NULL,
    modifiers_ids UUID[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE problems ADD COLUMN lang_overrides JSONB DEFAULT '[]'::jsonb;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF NOT EXISTS problems;
