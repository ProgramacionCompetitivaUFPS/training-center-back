-- +goose Up

-- GET /users/search y el filtro admin hacen name/nickname ILIKE '%…%', que un btree no puede usar.
-- pg_trgm ya existe desde la 035.
CREATE INDEX IF NOT EXISTS idx_users_name_trgm ON users USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_nickname_trgm ON users USING gin (nickname gin_trgm_ops);

-- validación de registro de equipos: selected_members && $n::uuid[] (operador de overlap).
CREATE INDEX IF NOT EXISTS idx_ctp_selected_members ON contest_team_participants USING gin (selected_members);

-- GET .../contests/{id}/submissions pagina en SQL por submitted_at DESC, id.
CREATE INDEX IF NOT EXISTS idx_submissions_contest_submitted
    ON submissions (contest_id, submitted_at DESC, id)
    WHERE contest_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_submissions_contest_submitted;
DROP INDEX IF EXISTS idx_ctp_selected_members;
DROP INDEX IF EXISTS idx_users_nickname_trgm;
DROP INDEX IF EXISTS idx_users_name_trgm;
