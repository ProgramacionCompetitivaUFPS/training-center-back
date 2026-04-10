-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL UNIQUE,
    country VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    institution VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'CONTESTANT',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    preferences JSONB,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT chk_active_user_email CHECK (
        status != 'ACTIVE' OR email IS NOT NULL
    )
);

-- +goose Down
DROP TABLE IF EXISTS users;
