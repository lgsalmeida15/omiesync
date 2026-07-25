ALTER TABLE _etl.usuario_grupos ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'viewer';

-- Popula com o role atual de cada usuário
UPDATE _etl.usuario_grupos ug
SET role = u.role
FROM _etl.usuarios u
WHERE ug.usuario_id = u.id;
