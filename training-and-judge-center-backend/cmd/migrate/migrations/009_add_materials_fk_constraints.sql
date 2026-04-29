-- +goose Up
ALTER TABLE materials
    ADD CONSTRAINT fk_materials_group_id  FOREIGN KEY (group_id)  REFERENCES groups(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_materials_author_id FOREIGN KEY (author_id) REFERENCES users(id),
    ADD CONSTRAINT chk_materials_status   CHECK (status IN ('DRAFT', 'PUBLISHED'));

-- +goose Down
ALTER TABLE materials
    DROP CONSTRAINT IF EXISTS chk_materials_status,
    DROP CONSTRAINT IF EXISTS fk_materials_author_id,
    DROP CONSTRAINT IF EXISTS fk_materials_group_id;
