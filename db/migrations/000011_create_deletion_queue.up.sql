-- Migration 000011: Fila de exclusão com carência de 30 dias
CREATE TABLE IF NOT EXISTS _etl.deletion_queue (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id  UUID        NOT NULL REFERENCES _etl.empresas(id) ON DELETE CASCADE,
    execute_at  TIMESTAMPTZ NOT NULL,
    executed    BOOLEAN     NOT NULL DEFAULT false,
    executed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deletion_queue_execute_at
    ON _etl.deletion_queue (execute_at)
    WHERE executed = false;
