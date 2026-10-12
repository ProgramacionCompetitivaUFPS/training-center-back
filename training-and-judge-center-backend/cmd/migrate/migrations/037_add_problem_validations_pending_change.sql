-- +goose Up

-- pending_change: lo que un cambio en caliente de testCases/checker quiere poner
-- en efecto una vez validado. NULL en las validaciones de publicación.
ALTER TABLE problem_validations
    ADD COLUMN pending_change JSONB;

-- +goose Down

ALTER TABLE problem_validations
    DROP COLUMN IF EXISTS pending_change;
