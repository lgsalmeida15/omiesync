CREATE TABLE IF NOT EXISTS _etl.omie_endpoint_config (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    modulo          VARCHAR(50) NOT NULL UNIQUE,  -- "clientes", "movimentos_financeiros", etc.
    endpoint_path   VARCHAR(100) NOT NULL,         -- "/geral/clientes/"
    action          VARCHAR(100) NOT NULL,         -- "ListarClientes"
    array_field     VARCHAR(100) NOT NULL,         -- "clientes_cadastro"
    page_size       INT         NOT NULL DEFAULT 50,
    ativo           BOOLEAN     NOT NULL DEFAULT true,
    ignorar_delta   BOOLEAN     NOT NULL DEFAULT false,
    notas           TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID        REFERENCES _etl.usuarios(id)
);

-- Seed: valores atuais de produção (ON CONFLICT para garantir idempotência)
INSERT INTO _etl.omie_endpoint_config
    (modulo, endpoint_path, action, array_field, page_size, ignorar_delta, notas)
VALUES
    ('clientes',               '/geral/clientes/',       'ListarClientes',       'clientes_cadastro',        50, false, NULL),
    ('categorias',             '/geral/categorias/',     'ListarCategorias',     'categoria_cadastro',        50, false, 'Array usa snake_case diferente dos demais'),
    ('departamentos',          '/geral/departamentos/',  'ListarDepartamentos',  'DepartamentoCadastro',      50, false, NULL),
    ('contas_correntes',       '/geral/contacorrente/',  'ListarContasCorrentes','ListarContasCorrentes',     50, true,  'Array e action têm o mesmo nome; sempre busca tudo'),
    ('contas_pagar',           '/financas/contapagar/',  'ListarContasPagar',    'conta_pagar_cadastro',      50, false, NULL),
    ('contas_receber',         '/financas/contareceber/','ListarContasReceber',  'conta_receber_cadastro',    50, false, NULL),
    ('movimentos_financeiros', '/financas/mf/',          'ListarMovimentos',     'movimentos',                50, true,  'Paginação especial: nPagina/nRegPorPagina; sempre busca tudo'),
    ('extrato',                '/financas/extrato/',     'ListarExtrato',        'listaMovimentos',           0,  false, 'Sem paginação padrão; subdivisão binária por período; 0 = sem pageSize'),
    ('ordens_servico',         '/servicos/os/',          'ListarOS',             'osCadastro',                50, false, NULL),
    ('projetos',               '/geral/projetos/',       'ListarProjetos',       'cadastro',                  50, false, 'Array field é genérico "cadastro"')
ON CONFLICT (modulo) DO NOTHING;
