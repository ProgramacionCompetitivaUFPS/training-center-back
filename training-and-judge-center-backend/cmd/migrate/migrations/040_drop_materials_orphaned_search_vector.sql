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
-- de construir o tirar un índice. "Breve" es cuánto tarda una vez que lo consigue, no
-- cuánto espera para conseguirlo: ACCESS EXCLUSIVE necesita que termine cualquier otra
-- transacción con un lock sobre materials, y mientras espera, encola detrás de sí toda
-- lectura/escritura nueva a la tabla (aunque esas sí serían compatibles entre ellas). Un
-- lock_timeout acotado hace que falle rápido y limpio en vez de colgarse indefinidamente
-- si hay algo más reteniendo la tabla — se reintenta la migración cuando esa transacción
-- termine.
DROP INDEX CONCURRENTLY IF EXISTS idx_materials_search;
SET lock_timeout = '5s';
ALTER TABLE materials DROP COLUMN IF EXISTS search_vector;
RESET lock_timeout;

-- +goose Down

-- Camino de vuelta, poco crítico (solo se usa si hay que revertir): el ADD COLUMN con
-- STORED reescribe la tabla completa y no admite CONCURRENTLY de ninguna forma, así que
-- esa parte no se puede hacer más segura — se deja tal cual. IF NOT EXISTS sí, para que
-- un reintento tras un fallo a medias no choque con "la columna ya existe".
ALTER TABLE materials ADD COLUMN IF NOT EXISTS search_vector TSVECTOR GENERATED ALWAYS AS (
    setweight(to_tsvector('english', title), 'A') ||
    setweight(to_tsvector('english', coalesce(content, '')), 'B')
) STORED;

-- Si un intento previo de CREATE INDEX CONCURRENTLY falló a medias, Postgres deja el
-- índice creado pero marcado INVALID — y como el nombre ya existe, IF NOT EXISTS lo
-- daría por "ya está" sin reconstruirlo nunca. Se tira primero sin condición (es idempotente
-- y barato: a lo sumo reconstruye uno que ya estaba bien) para que el intento siguiente
-- siempre arranque limpio.
DROP INDEX CONCURRENTLY IF EXISTS idx_materials_search;
CREATE INDEX CONCURRENTLY idx_materials_search ON materials USING GIN(search_vector);
