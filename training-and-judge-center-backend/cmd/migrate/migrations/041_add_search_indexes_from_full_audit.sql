-- +goose NO TRANSACTION
-- +goose Up

-- Barrido completo del backend (2026-10-17) buscando los mismos patrones ya corregidos en
-- la 038/039: ILIKE/LIKE sin índice trigram, y falta de un índice que calce con una consulta
-- paginada caliente. CONCURRENTLY + DROP previo, mismo motivo y mismo patrón que la 038/039/040
-- (contests/users/groups/submissions reciben escrituras en producción).

-- GET /contests?search=... (ListContests/ListMyContests) hace name ILIKE '%...%' — mismo
-- patrón que problems.title/users.name que ya tienen trigram, nunca se le aplicó a contests.
DROP INDEX CONCURRENTLY IF EXISTS idx_contests_name_trgm;
CREATE INDEX CONCURRENTLY idx_contests_name_trgm ON contests USING gin (name gin_trgm_ops);

-- GET /admin/users con searchField=email|institution|all hace ILIKE sobre estas dos columnas
-- (repository.go FindAll) — la 038 solo cubrió name/nickname.
DROP INDEX CONCURRENTLY IF EXISTS idx_users_email_trgm;
CREATE INDEX CONCURRENTLY idx_users_email_trgm ON users USING gin (email gin_trgm_ops);
DROP INDEX CONCURRENTLY IF EXISTS idx_users_institution_trgm;
CREATE INDEX CONCURRENTLY idx_users_institution_trgm ON users USING gin (institution gin_trgm_ops);

-- GET /groups?search=... hace LOWER(g.name) LIKE $n OR LOWER(COALESCE(g.description,'')) LIKE $n
-- (group/repository.go) — el índice que ya existe (idx_groups_name, UNIQUE sobre LOWER(name))
-- es para igualdad, no sirve para un LIKE con comodín al inicio. El índice trigram tiene que
-- calcar la MISMA expresión de la query para que el planner lo use.
DROP INDEX CONCURRENTLY IF EXISTS idx_groups_name_lower_trgm;
CREATE INDEX CONCURRENTLY idx_groups_name_lower_trgm ON groups USING gin (LOWER(name) gin_trgm_ops);
DROP INDEX CONCURRENTLY IF EXISTS idx_groups_description_lower_trgm;
CREATE INDEX CONCURRENTLY idx_groups_description_lower_trgm ON groups USING gin (LOWER(COALESCE(description, '')) gin_trgm_ops);

-- GET /problems/p/{slug}/submissions sin OnlyMine/Admin (el camino común: cualquiera viendo
-- los envíos públicos de un problema fuera de competencia) filtra
-- problem_id = $1 AND contest_id IS NULL AND visibility = $2 ORDER BY submitted_at DESC
-- (list_problem_submissions.go + submission/repository.go buildWhere). idx_submissions_problem_id
-- solo cubre la igualdad de problem_id; nada cubre visibility ni deja aprovechar el ORDER BY.
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_problem_public_submitted;
CREATE INDEX CONCURRENTLY idx_submissions_problem_public_submitted
    ON submissions (problem_id, visibility, submitted_at DESC)
    WHERE contest_id IS NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_problem_public_submitted;
DROP INDEX CONCURRENTLY IF EXISTS idx_groups_description_lower_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_groups_name_lower_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_institution_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_email_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_contests_name_trgm;
