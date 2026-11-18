-- +goose Up

-- Una inscripción de equipo siempre tiene al menos un integrante seleccionado: el mínimo del
-- contest ya es >= 1 y quien sale se descuenta con esa regla. Un arreglo vacío no puede existir.
-- Las filas vacías que hubiera (solo las dejaba el borrado de un miembro del grupo) no aportan a
-- nadie al ranking; se eliminan para que la restricción pueda aplicarse.
DELETE FROM contest_team_participants
WHERE cardinality(selected_members) = 0;

ALTER TABLE contest_team_participants
    ADD CONSTRAINT chk_ctp_selected_members_not_empty
        CHECK (cardinality(selected_members) >= 1);

-- +goose Down

ALTER TABLE contest_team_participants
    DROP CONSTRAINT IF EXISTS chk_ctp_selected_members_not_empty;
