-- Migration 000036: coleta de contratos de servico (/servicos/contrato/ ListarContratos).
--
-- Contexto: lancamentos do extrato originados de contrato nao tem titulo em
-- contas_pagar/contas_receber. Nas matvw gerenciais o join com cp_categorias /
-- cp_distribuicao falha e a linha degrada: categoria unica sem rateio, cliente
-- apenas como texto (cRazCliente) e 'Sem departamento'.
--
-- Esta migration so cria a base da coleta. O enriquecimento das matvw vem depois.
--
-- Chave de ligacao: cabecalho.nCodCtr (gravado em codigo_lancamento_omie) e
-- referenciado por extrato.raw ->> 'cNumero'. A API do ListarExtrato nao devolve
-- nenhum campo nCodCtr, entao numero_contrato (cNumCtr) fica ao lado para permitir
-- validar a chave nos dois formatos sem recoletar nada.
--
-- Espelha o DDL de internal/db/provisioner.go para schemas ja provisionados. O
-- provisioner tambem cria a tabela sozinho via auto-upgrade (CurrentSchemaVersion 11);
-- as duas rotas sao idempotentes e podem rodar em qualquer ordem.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        EXECUTE format($f$
            CREATE TABLE IF NOT EXISTS %I.contratos (
                id                     BIGSERIAL   PRIMARY KEY,
                empresa_id             UUID        NOT NULL,
                codigo_lancamento_omie BIGINT      NOT NULL,
                numero_contrato        TEXT,
                codigo_cliente         BIGINT,
                codigo_situacao        TEXT,
                vigencia_inicial       DATE,
                vigencia_final         DATE,
                dia_faturamento        INT,
                valor_total_mes        NUMERIC(15,2),
                tipo_faturamento       TEXT,
                codigo_categoria       TEXT,
                codigo_projeto         BIGINT,
                codigo_conta_corrente  BIGINT,
                raw                    JSONB,
                synced_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                UNIQUE(empresa_id, codigo_lancamento_omie)
            )
        $f$, r.schema_name);

        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_contratos_numero ON %I.contratos (empresa_id, numero_contrato)', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_contratos_cliente ON %I.contratos (empresa_id, codigo_cliente)', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_contratos_raw ON %I.contratos USING GIN (raw)', r.schema_name, r.schema_name);

    END LOOP;
END
$$;

-- Configuracao do endpoint. modulo precisa bater exatamente com ContratosExecutor.Nome().
-- page_size 50 e o default da API. ignorar_delta = false: o ListarContratos aceita
-- filtrar_por_data_de, entao o sync incremental funciona.
INSERT INTO _etl.omie_endpoint_config
    (modulo, endpoint_path, action, array_field, page_size, ignorar_delta, notas)
VALUES
    ('contratos', '/servicos/contrato/', 'ListarContratos', 'contratoCadastro', 50, false,
     'Contratos de servico. apenas_importado_api=N fixo no executor. Campos principais vem aninhados em "cabecalho". Chave de ligacao com o extrato: cabecalho.nCodCtr <-> extrato.raw->>cNumero.')
ON CONFLICT (modulo) DO NOTHING;
