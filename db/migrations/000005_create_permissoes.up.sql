-- Migration 000005: Permissões granulares por usuário × empresa × recurso × ação
CREATE TABLE IF NOT EXISTS _etl.permissoes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id  UUID        NOT NULL REFERENCES _etl.usuarios(id) ON DELETE CASCADE,
    empresa_id  UUID        NOT NULL REFERENCES _etl.empresas(id) ON DELETE CASCADE,
    recurso     TEXT        NOT NULL CHECK (recurso IN ('dashboard', 'sync', 'admin')),
    acao        TEXT        NOT NULL CHECK (acao IN ('ver', 'editar', 'forcar_sync')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (usuario_id, empresa_id, recurso, acao)
);

CREATE INDEX IF NOT EXISTS idx_permissoes_usuario_id ON _etl.permissoes (usuario_id);
CREATE INDEX IF NOT EXISTS idx_permissoes_empresa_id ON _etl.permissoes (empresa_id);
