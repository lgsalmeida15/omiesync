-- name: InsertGrupo :one
INSERT INTO _etl.grupos (nome, slug, schema_name, status)
VALUES ($1, $2, $3, 'ativo')
RETURNING *;

-- name: GetGrupoByID :one
SELECT id, nome, slug, schema_name, status, created_at, updated_at, deleted_at
FROM _etl.grupos
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetGrupoBySlug :one
SELECT id, nome, slug, schema_name, status, created_at, updated_at, deleted_at
FROM _etl.grupos
WHERE slug = $1
  AND deleted_at IS NULL;

-- name: ListGrupos :many
SELECT id, nome, slug, schema_name, status, created_at, updated_at, deleted_at
FROM _etl.grupos
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountGrupos :one
SELECT COUNT(*) FROM _etl.grupos WHERE deleted_at IS NULL;

-- name: UpdateGrupo :one
UPDATE _etl.grupos
SET nome = $2, updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteGrupo :exec
UPDATE _etl.grupos
SET deleted_at = NOW(), status = 'deletando', updated_at = NOW()
WHERE id = $1;

-- name: CountEmpresasAtivasByGrupo :one
SELECT COUNT(*) FROM _etl.empresas
WHERE grupo_id = $1
  AND status != 'inativa'
  AND deleted_at IS NULL;
