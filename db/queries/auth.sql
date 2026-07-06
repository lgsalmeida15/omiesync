-- name: GetUsuarioByEmail :one
SELECT id, grupo_id, nome, email, password, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUsuarioByID :one
SELECT id, grupo_id, nome, email, password, role, ativo, created_at, updated_at
FROM _etl.usuarios
WHERE id = $1
  AND deleted_at IS NULL;

-- name: InsertRefreshToken :one
INSERT INTO _etl.refresh_tokens (usuario_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT id, usuario_id, token, expires_at, revoked, created_at
FROM _etl.refresh_tokens
WHERE token = $1
  AND revoked = false
  AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE token = $1;

-- name: RevokeAllUserTokens :exec
UPDATE _etl.refresh_tokens
SET revoked = true
WHERE usuario_id = $1;
