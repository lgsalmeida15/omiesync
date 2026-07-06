-- Migration 000009: Controle de sincronização por empresa
CREATE TABLE IF NOT EXISTS _etl.sync_control (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id       UUID        NOT NULL UNIQUE REFERENCES _etl.empresas(id) ON DELETE CASCADE,
    ativo            BOOLEAN     NOT NULL DEFAULT true,
    intervalo_min    INT         NOT NULL DEFAULT 60,
    ultimo_sync_at   TIMESTAMPTZ,
    proximo_sync_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_control_proximo ON _etl.sync_control (proximo_sync_at)
    WHERE ativo = true;
