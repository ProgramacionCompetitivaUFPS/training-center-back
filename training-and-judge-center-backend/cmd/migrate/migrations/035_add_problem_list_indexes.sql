-- +goose Up

-- GET /problems pagina por created_at DESC; sin este índice ordena la tabla entera.
CREATE INDEX IF NOT EXISTS idx_problems_created_at ON problems (created_at DESC);

-- filtro por autor (GET /problems?author=…) y su COUNT(*), que sin índice recorre toda la tabla.
CREATE INDEX IF NOT EXISTS idx_problems_author_id ON problems (author_id);

-- la búsqueda por título es title ILIKE '%…%', que un btree no puede usar.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_problems_title_trgm ON problems USING gin (title gin_trgm_ops);

-- +goose Down

DROP INDEX IF EXISTS idx_problems_title_trgm;
DROP INDEX IF EXISTS idx_problems_author_id;
DROP INDEX IF EXISTS idx_problems_created_at;
