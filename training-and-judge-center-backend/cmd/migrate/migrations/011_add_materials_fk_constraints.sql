-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    ALTER TABLE materials ADD CONSTRAINT fk_materials_group_id FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN
    ALTER TABLE materials ADD CONSTRAINT fk_materials_author_id FOREIGN KEY (author_id) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ BEGIN
    ALTER TABLE materials ADD CONSTRAINT chk_materials_status CHECK (status IN ('DRAFT', 'PUBLISHED'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE materials
    DROP CONSTRAINT IF EXISTS chk_materials_status,
    DROP CONSTRAINT IF EXISTS fk_materials_author_id,
    DROP CONSTRAINT IF EXISTS fk_materials_group_id;
