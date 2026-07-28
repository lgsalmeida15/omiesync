-- Migration 000025: garante que a coluna role existe em usuario_grupos
-- e restaura os roles corretos de todos os usuários.
-- A migration 000024 foi registrada mas nunca executada (baseline logic).
-- Esta migration é totalmente idempotente.

ALTER TABLE _etl.usuario_grupos ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'viewer';

-- Restaura o role de cada usuário a partir de usuarios.role.
-- admin_global sempre preserva admin_global em todos os seus grupos.
UPDATE _etl.usuario_grupos ug
SET role = u.role
FROM _etl.usuarios u
WHERE ug.usuario_id = u.id;
