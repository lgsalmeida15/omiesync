CREATE TABLE IF NOT EXISTS _etl.sync_job_progress (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID        NOT NULL REFERENCES _etl.sync_jobs(id) ON DELETE CASCADE,
    executor        VARCHAR(50) NOT NULL,          -- "clientes", "movimentos_financeiros", etc.
    status          VARCHAR(20) NOT NULL DEFAULT 'aguardando',
                                                   -- aguardando | rodando | concluido | erro
    pagina_atual    INT,
    total_paginas   INT,
    registros_proc  INT         NOT NULL DEFAULT 0,
    registros_total INT,                           -- preenchido quando Omie retorna total
    erro            TEXT,
    iniciado_at     TIMESTAMPTZ,
    concluido_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índice principal: buscar todo o progresso de um job
CREATE INDEX IF NOT EXISTS idx_sync_job_progress_job_id
    ON _etl.sync_job_progress (job_id);

-- Índice secundário: buscar progresso via empresa (join com sync_jobs)
CREATE INDEX IF NOT EXISTS idx_sync_job_progress_updated
    ON _etl.sync_job_progress (job_id, updated_at DESC);
