-- Migration 000002: Tabela de grupos (tenants)
-- UP
CREATE TABLE IF NOT EXISTS _etl.grupos (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nome        TEXT        NOT NULL,
    slug        TEXT        NOT NULL UNIQUE,
    schema_name TEXT        NOT NULL UNIQUE,
    status      TEXT        NOT NULL DEFAULT 'ativo' CHECK (status IN ('ativo', 'inativo', 'deletando')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_grupos_slug ON _etl.grupos (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_grupos_status ON _etl.grupos (status) WHERE deleted_at IS NULL;
