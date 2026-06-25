-- +goose Up

CREATE INDEX IF NOT EXISTS idx_materials_fts
    ON materials USING GIN (
        (setweight(to_tsvector('simple', title), 'A') ||
         setweight(to_tsvector('simple', COALESCE(content, '')), 'B'))
    );

-- +goose Down

DROP INDEX IF EXISTS idx_materials_fts;
