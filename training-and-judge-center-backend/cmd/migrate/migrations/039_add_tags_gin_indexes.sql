-- +goose NO TRANSACTION
-- +goose Up

-- CONCURRENTLY: materials y problems reciben escrituras en producción (crear/editar
-- material o problema) — ver la 038 para el porqué completo. NO TRANSACTION de arriba
-- aplica a todo el archivo.
--
-- GET /groups/{groupId}/materials?tags=... y GET /problems?tags=... filtran con el
-- operador de contención de arreglo (tags @> $n), que un btree no puede usar — mismo
-- patrón que selected_members en la 038. materials.tags está acotado por group_id,
-- pero el grupo "Global" (todos los usuarios son miembros, ver 008_seed_global_group)
-- no tiene el mismo techo que un grupo de curso normal. problems.tags no tiene ningún
-- acotador: la tabla es única para toda la plataforma, sin group_id, y el frontend la
-- expone con un filtro de tags visible (ProblemsPage.tsx).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_materials_tags ON materials USING gin (tags);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problems_tags  ON problems  USING gin (tags);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_problems_tags;
DROP INDEX CONCURRENTLY IF EXISTS idx_materials_tags;
