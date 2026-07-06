CREATE TABLE IF NOT EXISTS _etl.sync_job_pages (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id            UUID        NOT NULL REFERENCES _etl.sync_jobs(id) ON DELETE CASCADE,
    modulo            VARCHAR(50) NOT NULL,
    pagina            INT         NOT NULL,
    total_paginas     INT         NOT NULL DEFAULT 1,

    status            VARCHAR(20) NOT NULL DEFAULT 'pendente',
    -- valores permitidos: pendente | rodando | concluido | erro | cancelado

    tentativas        INT         NOT NULL DEFAULT 0,
    max_tentativas    INT         NOT NULL DEFAULT 3,
    proximo_retry_at  TIMESTAMPTZ,

    registros_gravados INT        NOT NULL DEFAULT 0,
    erro               TEXT,

    agendado_para      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    iniciado_at        TIMESTAMPTZ,
    concluido_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_sync_job_pages_status
        CHECK (status IN ('pendente', 'rodando', 'concluido', 'erro', 'cancelado')),
    CONSTRAINT chk_sync_job_pages_pagina
        CHECK (pagina >= 1),
    CONSTRAINT chk_sync_job_pages_tentativas
        CHECK (tentativas >= 0 AND tentativas <= max_tentativas + 1)
);

-- Índice para buscar páginas pendentes de um job (usado pelo page worker)
CREATE INDEX idx_sync_job_pages_pending
    ON _etl.sync_job_pages (job_id, status, proximo_retry_at)
    WHERE status IN ('pendente', 'erro');

-- Índice para buscar todas as páginas de um job (usado pelo drawer)
CREATE INDEX idx_sync_job_pages_job
    ON _etl.sync_job_pages (job_id, modulo, pagina);
