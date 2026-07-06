-- Migration 000003: Tabela de empresas por grupo
CREATE TABLE IF NOT EXISTS _etl.empresas (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    grupo_id        UUID        NOT NULL REFERENCES _etl.grupos(id) ON DELETE CASCADE,
    nome            TEXT        NOT NULL,
    cnpj            TEXT,
    app_key         TEXT        NOT NULL,
    app_secret      TEXT        NOT NULL,
    status_sync     TEXT        NOT NULL DEFAULT 'ativo' CHECK (status_sync IN ('ativo', 'pausado', 'erro', 'deletando')),
    status          TEXT        NOT NULL DEFAULT 'ativa' CHECK (status IN ('ativa', 'inativa', 'deletando')),
    ultimo_sync_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_empresas_grupo_id ON _etl.empresas (grupo_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_empresas_status   ON _etl.empresas (status)   WHERE deleted_at IS NULL;
