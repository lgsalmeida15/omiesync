-- name: InsertEmpresa :one
INSERT INTO _etl.empresas (grupo_id, nome, cnpj, app_key, app_secret, status, status_sync)
VALUES ($1, $2, $3, $4, $5, 'ativa', 'ativo')
RETURNING id, grupo_id, nome, cnpj, app_key, app_secret, status, status_sync, ultimo_sync_at, created_at, updated_at;

-- name: GetEmpresaByID :one
SELECT id, grupo_id, nome, cnpj, app_key, app_secret, status, status_sync, ultimo_sync_at, created_at, updated_at
FROM _etl.empresas
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListEmpresasByGrupo :many
SELECT id, grupo_id, nome, cnpj, app_key, app_secret, status, status_sync, ultimo_sync_at, created_at, updated_at
FROM _etl.empresas
WHERE grupo_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEmpresasByGrupo :one
SELECT COUNT(*) FROM _etl.empresas
WHERE grupo_id = $1
  AND deleted_at IS NULL;

-- name: UpdateEmpresa :one
UPDATE _etl.empresas
SET nome = $2, cnpj = $3, app_key = $4, app_secret = $5, updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, grupo_id, nome, cnpj, app_key, app_secret, status, status_sync, ultimo_sync_at, created_at, updated_at;

-- name: MarkEmpresaDeletando :exec
UPDATE _etl.empresas
SET status = 'deletando', status_sync = 'pausado', deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: InsertDeletionQueue :one
INSERT INTO _etl.deletion_queue (empresa_id, execute_at)
VALUES ($1, $2)
RETURNING *;

-- name: ReativarEmpresa :exec
UPDATE _etl.empresas
SET status = 'ativa', status_sync = 'ativo', deleted_at = NULL, updated_at = NOW()
WHERE id = $1;

-- name: CancelDeletionQueue :exec
UPDATE _etl.deletion_queue
SET executed = true, executed_at = NOW()
WHERE empresa_id = $1
  AND executed = false;

-- name: ListPendingDeletions :many
SELECT id, empresa_id, execute_at, executed, executed_at, created_at
FROM _etl.deletion_queue
WHERE executed = false
  AND execute_at <= NOW()
ORDER BY execute_at ASC;

-- name: MarkDeletionExecuted :exec
UPDATE _etl.deletion_queue
SET executed = true, executed_at = NOW()
WHERE id = $1;
