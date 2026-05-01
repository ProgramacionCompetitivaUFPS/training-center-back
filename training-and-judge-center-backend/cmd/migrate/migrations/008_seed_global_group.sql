-- +goose Up

-- Usuario sistema: referencia interna para entidades de sistema.
-- status=DEACTIVATED + email=NULL pasa el CHECK constraint de users
-- (que exige email solo cuando status='ACTIVE').
-- ON CONFLICT DO NOTHING garantiza idempotencia en re-ejecuciones.
INSERT INTO users (
    id, email, password, name, nickname,
    country, city, institution, role, status, created_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    NULL,
    '$SYSTEM_NO_LOGIN$',
    'System',
    'system',
    'N/A',
    'N/A',
    'N/A',
    'ADMIN',
    'DEACTIVATED',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Global Group: grupo por defecto del sistema.
-- is_default=TRUE activa el unique partial index idx_groups_single_default,
-- garantizando que solo exista un grupo default por instalación.
INSERT INTO groups (
    id, name, description, visibility, join_policy,
    is_default, created_by, created_at, updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000002',
    'Global',
    'The default group. All members are automatically part of this group.',
    'VISIBLE',
    'OPEN',
    TRUE,
    '00000000-0000-0000-0000-000000000001',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- +goose Down

DELETE FROM groups WHERE id = '00000000-0000-0000-0000-000000000002';
DELETE FROM users  WHERE id = '00000000-0000-0000-0000-000000000001';
