-- Migração 000016: Configuração de executors por empresa e sync seletivo
CREATE TABLE IF NOT EXISTS _etl.empresa_executor_config (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id  UUID        NOT NULL REFERENCES _etl.empresas(id) ON DELETE CASCADE,
    executor    VARCHAR(50) NOT NULL,
    ativo       BOOLEAN     NOT NULL DEFAULT true,
    notas       TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by  UUID        REFERENCES _etl.usuarios(id),
    UNIQUE (empresa_id, executor)
);

CREATE INDEX IF NOT EXISTS idx_empresa_executor_config_empresa
    ON _etl.empresa_executor_config (empresa_id);

ALTER TABLE _etl.sync_jobs
    ADD COLUMN IF NOT EXISTS executor VARCHAR(50);
