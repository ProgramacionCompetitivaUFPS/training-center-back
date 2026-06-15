-- +goose Up
ALTER TABLE join_requests
    DROP CONSTRAINT join_requests_group_id_fkey,
    ADD CONSTRAINT join_requests_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE join_requests
    DROP CONSTRAINT join_requests_group_id_fkey,
    ADD CONSTRAINT join_requests_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id);
