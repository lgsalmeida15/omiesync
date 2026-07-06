-- Migration 000004: Tabela de usuários
CREATE TABLE IF NOT EXISTS _etl.usuarios (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    grupo_id    UUID        REFERENCES _etl.grupos(id) ON DELETE CASCADE,
    nome        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    password    TEXT        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin_global', 'admin_grupo', 'viewer')),
    ativo       BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_usuarios_email    ON _etl.usuarios (email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_usuarios_grupo_id ON _etl.usuarios (grupo_id) WHERE deleted_at IS NULL;
