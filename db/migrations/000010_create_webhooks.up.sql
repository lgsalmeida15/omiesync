-- Migration 000010: Configuração de webhooks por grupo
CREATE TABLE IF NOT EXISTS _etl.webhooks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    grupo_id    UUID        NOT NULL REFERENCES _etl.grupos(id) ON DELETE CASCADE,
    url         TEXT        NOT NULL,
    secret      TEXT        NOT NULL,
    eventos     TEXT[]      NOT NULL DEFAULT '{}',
    ativo       BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_grupo_id ON _etl.webhooks (grupo_id) WHERE ativo = true;
