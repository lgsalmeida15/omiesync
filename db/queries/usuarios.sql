-- name: InsertUsuario :one
INSERT INTO _etl.usuarios (grupo_id, nome, email, password, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, grupo_id, nome, email, role, ativo, created_at, updated_at;

-- name: GetUsuarioByIDFull :one
SELECT id, grupo_id, nome, email, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUsuariosByGrupo :many
SELECT id, grupo_id, nome, email, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE grupo_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsuariosByGrupo :one
SELECT COUNT(*) FROM _etl.usuarios
WHERE grupo_id = $1
  AND deleted_at IS NULL;

-- name: UpdateUsuario :one
UPDATE _etl.usuarios
SET nome = $2, role = $3, ativo = $4, updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, grupo_id, nome, email, role, ativo, created_at, updated_at;

-- name: UpdateUsuarioPassword :exec
UPDATE _etl.usuarios
SET password = $2, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUsuario :exec
UPDATE _etl.usuarios
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;
