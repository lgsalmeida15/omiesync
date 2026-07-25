-- Tabela N:N para suporte a usuários em múltiplos grupos
CREATE TABLE IF NOT EXISTS _etl.usuario_grupos (
    usuario_id UUID NOT NULL REFERENCES _etl.usuarios(id) ON DELETE CASCADE,
    grupo_id   UUID NOT NULL REFERENCES _etl.grupos(id)   ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (usuario_id, grupo_id)
);

CREATE INDEX IF NOT EXISTS idx_usuario_grupos_usuario ON _etl.usuario_grupos (usuario_id);
CREATE INDEX IF NOT EXISTS idx_usuario_grupos_grupo   ON _etl.usuario_grupos (grupo_id);

-- Migrar vínculos existentes sem perda de dados
INSERT INTO _etl.usuario_grupos (usuario_id, grupo_id)
SELECT id, grupo_id
FROM _etl.usuarios
WHERE grupo_id IS NOT NULL
ON CONFLICT DO NOTHING;
