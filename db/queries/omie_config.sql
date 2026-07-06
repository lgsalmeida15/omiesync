-- name: ListOmieEndpointConfigs :many
SELECT c.id, c.modulo, c.endpoint_path, c.action, c.array_field, c.page_size, 
       c.ativo, c.ignorar_delta, c.notas, c.updated_at, c.updated_by,
       u.email as updated_by_email
FROM _etl.omie_endpoint_config c
LEFT JOIN _etl.usuarios u ON u.id = c.updated_by
ORDER BY c.modulo ASC;

-- name: GetOmieEndpointConfigByModulo :one
SELECT c.id, c.modulo, c.endpoint_path, c.action, c.array_field, c.page_size, 
       c.ativo, c.ignorar_delta, c.notas, c.updated_at, c.updated_by,
       u.email as updated_by_email
FROM _etl.omie_endpoint_config c
LEFT JOIN _etl.usuarios u ON u.id = c.updated_by
WHERE c.modulo = $1;

-- name: UpdateOmieEndpointConfig :one
UPDATE _etl.omie_endpoint_config
SET endpoint_path = $2,
    action = $3,
    array_field = $4,
    page_size = $5,
    ativo = $6,
    notas = $7,
    updated_at = NOW(),
    updated_by = $8
WHERE modulo = $1
RETURNING *;
