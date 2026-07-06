-- name: ListWebhooksByGrupo :many
SELECT id, grupo_id, url, secret, eventos, ativo, created_at, updated_at
FROM _etl.webhooks
WHERE grupo_id = $1
  AND ativo = true;

-- name: InsertWebhook :one
INSERT INTO _etl.webhooks (grupo_id, url, secret, eventos)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM _etl.webhooks WHERE id = $1;
