-- +goose NO TRANSACTION
-- +goose Up

-- search_vector (migración 006, generada con to_tsvector('english', ...)) quedó huérfana
-- cuando la 026 introdujo idx_materials_fts con su propia expresión en caliente usando
-- configuración 'simple' (necesaria para tokenizar títulos como "Node.js" igual que la
-- búsqueda en repository.go). Ninguna query del código selecciona search_vector ni usa
-- idx_materials_search — solo pagan el costo de recalcularla y reindexarla en cada
-- INSERT/UPDATE de un material, sin que ninguna lectura la use.
--
-- DROP INDEX CONCURRENTLY por la misma razón que en la 038/039 (materials recibe
-- escrituras). DROP COLUMN no lleva CONCURRENTLY — no existe para eso: es un cambio de
-- catálogo, el lock que toma es breve sin importar el tamaño de la tabla, a diferencia
-- de construir o tirar un índice.
DROP INDEX CONCURRENTLY IF EXISTS idx_materials_search;
ALTER TABLE materials DROP COLUMN IF EXISTS search_vector;

-- +goose Down

-- Camino de vuelta, poco crítico (solo se usa si hay que revertir): el ADD COLUMN con
-- STORED reescribe la tabla completa y no admite CONCURRENTLY de ninguna forma, así que
-- esa parte no se puede hacer más segura — se deja tal cual.
ALTER TABLE materials ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    setweight(to_tsvector('english', title), 'A') ||
    setweight(to_tsvector('english', coalesce(content, '')), 'B')
) STORED;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_materials_search ON materials USING GIN(search_vector);
