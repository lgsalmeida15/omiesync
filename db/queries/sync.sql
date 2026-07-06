-- name: InsertSyncJob :one
INSERT INTO _etl.sync_jobs (empresa_id, tipo, status, executor)
VALUES ($1, $2, 'pendente', $3)
RETURNING *;

-- name: GetSyncJobByID :one
SELECT id, empresa_id, tipo, status, erro, iniciado_at, concluido_at, created_at, executor, ultimo_heartbeat_at
FROM _etl.sync_jobs
WHERE id = $1;

-- name: ListSyncJobsByEmpresa :many
SELECT id, empresa_id, tipo, status, erro, iniciado_at, concluido_at, created_at, executor, ultimo_heartbeat_at
FROM _etl.sync_jobs
WHERE empresa_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSyncJobsByEmpresa :one
SELECT COUNT(*) FROM _etl.sync_jobs WHERE empresa_id = $1;

-- name: UpdateSyncJobStatus :one
UPDATE _etl.sync_jobs
SET status = $2, erro = $3, iniciado_at = $4, concluido_at = $5
WHERE id = $1
RETURNING *;

-- name: GetSyncControl :one
SELECT id, empresa_id, ativo, intervalo_incremental_min, intervalo_full_dias, 
       ultimo_sync_at, proximo_sync_at, ultimo_full_sync_at, proximo_full_sync_at, 
       created_at, updated_at
FROM _etl.sync_control
WHERE empresa_id = $1;

-- name: UpsertSyncControl :one
INSERT INTO _etl.sync_control (
    empresa_id, ativo, intervalo_incremental_min, intervalo_full_dias, 
    proximo_sync_at, proximo_full_sync_at
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (empresa_id) DO UPDATE
    SET ativo                     = EXCLUDED.ativo,
        intervalo_incremental_min = EXCLUDED.intervalo_incremental_min,
        intervalo_full_dias       = EXCLUDED.intervalo_full_dias,
        proximo_sync_at           = EXCLUDED.proximo_sync_at,
        proximo_full_sync_at      = EXCLUDED.proximo_full_sync_at,
        updated_at                = NOW()
RETURNING *;

-- name: UpdateSyncControlAfterRun :exec
UPDATE _etl.sync_control
SET ultimo_sync_at  = CASE WHEN sqlc.arg('tipo')::text != 'full' THEN NOW() ELSE ultimo_sync_at END,
    ultimo_full_sync_at = CASE WHEN sqlc.arg('tipo')::text = 'full' THEN NOW() ELSE ultimo_full_sync_at END,
    updated_at      = NOW()
WHERE empresa_id = $1;

-- name: AdvanceSyncScheduleOnDispatch :exec
UPDATE _etl.sync_control
SET proximo_sync_at = CASE 
        WHEN sqlc.arg('tipo')::text != 'full' THEN proximo_sync_at + (intervalo_incremental_min * INTERVAL '1 minute') 
        ELSE proximo_sync_at 
    END,
    proximo_full_sync_at = CASE 
        WHEN sqlc.arg('tipo')::text = 'full' THEN proximo_full_sync_at + (intervalo_full_dias * INTERVAL '1 day') 
        ELSE proximo_full_sync_at 
    END,
    updated_at = NOW()
WHERE empresa_id = $1;

-- name: GetEmpresasDueForIncremental :many
SELECT sc.empresa_id
FROM _etl.sync_control sc
JOIN _etl.empresas e ON e.id = sc.empresa_id
WHERE sc.ativo = true
  AND sc.proximo_sync_at <= NOW()
  AND e.status = 'ativa'
  AND e.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM _etl.sync_jobs sj
    WHERE sj.empresa_id = sc.empresa_id
      AND sj.status IN ('pendente', 'rodando')
  );

-- name: GetEmpresasDueForFull :many
SELECT sc.empresa_id
FROM _etl.sync_control sc
JOIN _etl.empresas e ON e.id = sc.empresa_id
WHERE sc.ativo = true
  AND sc.proximo_full_sync_at <= NOW()
  AND e.status = 'ativa'
  AND e.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM _etl.sync_jobs sj
    WHERE sj.empresa_id = sc.empresa_id
      AND sj.status IN ('pendente', 'rodando')
  );

-- name: GetSyncJobProgress :many
SELECT executor, status, pagina_atual, total_paginas, registros_proc, registros_total, 
       erro, iniciado_at, concluido_at, updated_at,
       ultimo_payload, ultimo_response, erro_payload, erro_response
FROM _etl.sync_job_progress
WHERE job_id = $1
ORDER BY iniciado_at ASC NULLS LAST, updated_at ASC;

-- name: GetExecutorConfigsByEmpresa :many
SELECT executor, ativo, notas, updated_at, updated_by
FROM _etl.empresa_executor_config
WHERE empresa_id = $1;

-- name: UpsertExecutorConfig :one
INSERT INTO _etl.empresa_executor_config (empresa_id, executor, ativo, notas, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (empresa_id, executor) DO UPDATE
    SET ativo      = EXCLUDED.ativo,
        notas      = EXCLUDED.notas,
        updated_by = EXCLUDED.updated_by,
        updated_at = NOW()
RETURNING *;

-- name: GetEnabledExecutorsByEmpresa :many
SELECT executor
FROM _etl.empresa_executor_config
WHERE empresa_id = $1 AND ativo = false;

-- name: GetJobAtivo :one
SELECT COUNT(*) OVER()::int AS total,
       id,
       tipo,
       status,
       iniciado_at
FROM _etl.sync_jobs
WHERE empresa_id = $1
  AND status IN ('pendente', 'rodando')
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkStaleJobs :execrows
UPDATE _etl.sync_jobs
SET
    status      = 'erro',
    erro        = 'interrompido por reinício do servidor',
    concluido_at = NOW()
WHERE status IN ('rodando', 'pendente')
  AND concluido_at IS NULL;

-- name: UpdateJobHeartbeat :exec
UPDATE _etl.sync_jobs
SET ultimo_heartbeat_at = NOW()
WHERE id = @job_id;

-- name: GetJobsOverview :many
SELECT
    status,
    COUNT(*) AS total
FROM _etl.sync_jobs
WHERE concluido_at IS NULL OR status IN ('rodando', 'pendente')
GROUP BY status;

-- name: GetJobsAtivos :many
SELECT
    j.id,
    j.empresa_id,
    j.tipo,
    j.status,
    j.iniciado_at,
    j.ultimo_heartbeat_at,
    e.nome AS empresa_nome,
    g.nome AS grupo_nome,
    CASE
        WHEN j.status = 'rodando'
         AND j.ultimo_heartbeat_at IS NOT NULL
         AND j.ultimo_heartbeat_at < NOW() - INTERVAL '10 minutes'
        THEN true
        ELSE false
    END AS is_zumbi
FROM _etl.sync_jobs j
JOIN _etl.empresas e ON e.id = j.empresa_id
JOIN _etl.grupos g ON g.id = e.grupo_id
WHERE j.status IN ('rodando', 'pendente')
  AND j.concluido_at IS NULL
ORDER BY j.iniciado_at ASC;

-- name: CancelarJob :exec
UPDATE _etl.sync_jobs
SET
    status       = 'cancelado',
    erro         = 'cancelado manualmente pelo administrador',
    concluido_at = NOW()
WHERE id = @job_id
  AND status IN ('rodando', 'pendente');

-- name: InsertJobPage :exec
INSERT INTO _etl.sync_job_pages
    (job_id, modulo, pagina, total_paginas, status, agendado_para)
VALUES
    (@job_id, @modulo, @pagina, @total_paginas, 'pendente', NOW());

-- name: GetPendingPages :many
SELECT id, job_id, modulo, pagina, total_paginas, tentativas, max_tentativas, proximo_retry_at
FROM _etl.sync_job_pages
WHERE job_id = @job_id
  AND status IN ('pendente', 'erro')
  AND (proximo_retry_at IS NULL OR proximo_retry_at <= NOW())
  AND tentativas < max_tentativas
ORDER BY modulo, pagina
LIMIT @limit_count;

-- name: CountPendingPages :one
SELECT COUNT(*) FROM _etl.sync_job_pages
WHERE job_id = @job_id
  AND status NOT IN ('concluido', 'cancelado', 'erro');

-- name: ClaimPageForProcessing :one
UPDATE _etl.sync_job_pages
SET
    status      = 'rodando',
    iniciado_at = NOW(),
    tentativas  = tentativas + 1
WHERE id = @page_id
  AND status IN ('pendente', 'erro')
  AND (proximo_retry_at IS NULL OR proximo_retry_at <= NOW())
RETURNING id, job_id, modulo, pagina, total_paginas, tentativas, max_tentativas;

-- name: MarkPageConcluido :exec
UPDATE _etl.sync_job_pages
SET
    status             = 'concluido',
    concluido_at       = NOW(),
    registros_gravados = @registros_gravados,
    erro               = NULL
WHERE id = @page_id;

-- name: MarkPageErro :exec
UPDATE _etl.sync_job_pages
SET
    status           = 'erro',
    erro             = @erro,
    proximo_retry_at = @proximo_retry_at,
    concluido_at     = NULL
WHERE id = @page_id;

-- name: MarkPageCancelado :exec
UPDATE _etl.sync_job_pages
SET status = 'cancelado', concluido_at = NOW()
WHERE job_id = @job_id AND status IN ('pendente', 'erro');

-- name: GetDLQPages :many
SELECT
    p.id,
    p.job_id,
    p.modulo,
    p.pagina,
    p.total_paginas,
    p.tentativas,
    p.max_tentativas,
    p.erro,
    p.concluido_at,
    e.nome AS empresa_nome,
    g.nome AS grupo_nome
FROM _etl.sync_job_pages p
JOIN _etl.sync_jobs j ON j.id = p.job_id
JOIN _etl.empresas e ON e.id = j.empresa_id
JOIN _etl.grupos g ON g.id = e.grupo_id
WHERE p.status = 'erro'
  AND p.tentativas >= p.max_tentativas
ORDER BY p.concluido_at DESC
LIMIT 100;

-- name: RetryDLQPage :exec
UPDATE _etl.sync_job_pages
SET
    status           = 'pendente',
    tentativas       = 0,
    erro             = NULL,
    proximo_retry_at = NULL,
    concluido_at     = NULL
WHERE id = @page_id
  AND status = 'erro'
  AND tentativas >= max_tentativas;

-- name: GetPagesByJob :many
SELECT
    p.id,
    p.modulo,
    p.pagina,
    p.total_paginas,
    p.status,
    p.tentativas,
    p.max_tentativas,
    p.registros_gravados,
    p.erro,
    p.proximo_retry_at,
    p.iniciado_at,
    p.concluido_at
FROM _etl.sync_job_pages p
WHERE p.job_id = @job_id
ORDER BY p.modulo, p.pagina;

-- name: GetLatestJobIDByEmpresa :one
SELECT id FROM _etl.sync_jobs
WHERE empresa_id = @empresa_id
ORDER BY created_at DESC
LIMIT 1;

-- name: InitSyncJobProgress :exec
INSERT INTO _etl.sync_job_progress (job_id, executor, status, updated_at)
VALUES ($1, $2, 'aguardando', NOW())
ON CONFLICT DO NOTHING;

-- name: StartSyncJobProgress :exec
UPDATE _etl.sync_job_progress
SET status = 'rodando', iniciado_at = NOW(), updated_at = NOW()
WHERE job_id = $1 AND executor = $2;

-- name: UpdateSyncJobProgressPage :exec
UPDATE _etl.sync_job_progress
SET pagina_atual = $3, total_paginas = $4, registros_proc = $5, registros_total = $6, 
    ultimo_payload = $7, ultimo_response = $8, updated_at = NOW()
WHERE job_id = $1 AND executor = $2;

-- name: DoneSyncJobProgress :exec
UPDATE _etl.sync_job_progress
SET status = 'concluido', concluido_at = NOW(), registros_proc = $3, updated_at = NOW()
WHERE job_id = $1 AND executor = $2;

-- name: FailSyncJobProgress :exec
UPDATE _etl.sync_job_progress
SET status = 'erro', concluido_at = NOW(), erro = $3, 
    erro_payload = $4, erro_response = $5, updated_at = NOW()
WHERE job_id = $1 AND executor = $2;

-- name: SkipSyncJobProgress :exec
UPDATE _etl.sync_job_progress
SET status = 'pulado', erro = $3, updated_at = NOW()
WHERE job_id = $1 AND executor = $2;
