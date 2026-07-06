-- Migration 000006: Tabela de refresh tokens
CREATE TABLE IF NOT EXISTS _etl.refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id  UUID        NOT NULL REFERENCES _etl.usuarios(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token      ON _etl.refresh_tokens (token) WHERE revoked = false;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_usuario_id ON _etl.refresh_tokens (usuario_id);
