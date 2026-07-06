-- name: InsertAuditLog :one
INSERT INTO _etl.audit_logs (
    request_id,
    user_id,
    user_email,
    role,
    method,
    path,
    query_params,
    status_code,
    request_body,
    response_body,
    ip_address,
    user_agent,
    duration_ms
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;
