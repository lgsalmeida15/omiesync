-- Migration 000008: Jobs de sincronização
CREATE TABLE IF NOT EXISTS _etl.sync_jobs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id   UUID        NOT NULL REFERENCES _etl.empresas(id) ON DELETE CASCADE,
    tipo         TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'rodando', 'concluido', 'erro')),
    erro         TEXT,
    iniciado_at  TIMESTAMPTZ,
    concluido_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_jobs_empresa_id ON _etl.sync_jobs (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_status     ON _etl.sync_jobs (status);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_created_at ON _etl.sync_jobs (created_at DESC);
