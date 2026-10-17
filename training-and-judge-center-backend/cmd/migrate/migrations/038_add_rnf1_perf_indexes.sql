-- +goose NO TRANSACTION
-- +goose Up

-- CONCURRENTLY: estas tablas (users, contest_team_participants, submissions) reciben
-- escrituras en producción (login/registro, registro de equipos, envíos); un CREATE INDEX
-- normal toma un lock que bloquea esas escrituras mientras dura el build. CONCURRENTLY evita
-- ese lock a cambio de no poder correr dentro de una transacción — por eso el NO TRANSACTION
-- de arriba, que aplica a todo el archivo (Up y Down).
--
-- GET /users/search y el filtro admin hacen name/nickname ILIKE '%…%', que un btree no puede usar.
-- pg_trgm ya existe desde la 035.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_name_trgm ON users USING gin (name gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_nickname_trgm ON users USING gin (nickname gin_trgm_ops);

-- validación de registro de equipos: selected_members && $n::uuid[] (operador de overlap).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ctp_selected_members ON contest_team_participants USING gin (selected_members);

-- GET .../contests/{id}/submissions pagina en SQL por submitted_at DESC, id.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_contest_submitted
    ON submissions (contest_id, submitted_at DESC, id)
    WHERE contest_id IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_contest_submitted;
DROP INDEX CONCURRENTLY IF EXISTS idx_ctp_selected_members;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_nickname_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_name_trgm;
