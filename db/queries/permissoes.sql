-- name: InsertPermissao :one
INSERT INTO _etl.permissoes (usuario_id, empresa_id, recurso, acao)
VALUES ($1, $2, $3, $4)
ON CONFLICT (usuario_id, empresa_id, recurso, acao)
DO UPDATE SET created_at = _etl.permissoes.created_at
RETURNING *;

-- name: ListPermissoesByUsuario :many
SELECT id, usuario_id, empresa_id, recurso, acao, created_at
FROM _etl.permissoes
WHERE usuario_id = $1;

-- name: ListPermissoesByEmpresa :many
SELECT id, usuario_id, empresa_id, recurso, acao, created_at
FROM _etl.permissoes
WHERE empresa_id = $1;

-- name: DeletePermissao :exec
DELETE FROM _etl.permissoes
WHERE usuario_id = $1
  AND empresa_id = $2
  AND recurso    = $3
  AND acao       = $4;

-- name: HasPermissao :one
SELECT COUNT(*) FROM _etl.permissoes
WHERE usuario_id = $1
  AND empresa_id = $2
  AND recurso    = $3
  AND acao       = $4;
