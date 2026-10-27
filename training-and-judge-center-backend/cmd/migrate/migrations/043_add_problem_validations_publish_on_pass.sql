-- +goose Up

-- publish_on_pass: si publicar el problema al pasar la validación. TRUE en las validaciones que
-- vienen de "Publicar" (lo que hacían todas hasta ahora); FALSE en las de "Validar", que solo
-- comprueban el borrador y dejan la decisión de publicar al autor.
ALTER TABLE problem_validations
    ADD COLUMN publish_on_pass BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down

ALTER TABLE problem_validations
    DROP COLUMN IF EXISTS publish_on_pass;
