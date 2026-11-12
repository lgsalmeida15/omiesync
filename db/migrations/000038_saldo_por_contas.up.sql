-- Migration 000038: saldo real por conta corrente no dia 1 de cada mes
-- (/financas/extrato/ ListarExtrato, com cExibirApenasSaldo=S).
--
-- Contexto: contas_correntes.saldo_inicial e campo de cadastro do Omie SEM DATA
-- associada (o cadastro traz, em contas_correntes.raw ->> 'saldo_data', a data
-- real a que aquele saldo se refere, mas o codigo Go nunca leu esse campo). O
-- Resultado Acumulado usava esse saldo como se fosse sempre 01/01, acumulando o
-- resultado do ano por cima em cadeia — se a conta e antiga e o saldo de
-- cadastro e de uma data anterior, o erro de abertura se propaga e cresce o
-- ano inteiro.
--
-- saldo_por_contas guarda o saldo REAL (nSaldoAtual) de cada conta no dia 1 de
-- cada mes, direto do Omie. O calculo passa a ser independente por mes:
-- Acumulado(mes) = saldo_por_contas do mes + Resultado(mes), sem encadear com
-- o mes anterior — ver internal/dados/dashboard.go.
--
-- Espelha o DDL de internal/db/provisioner.go para schemas ja provisionados. O
-- provisioner tambem cria a tabela sozinho via auto-upgrade (CurrentSchemaVersion 13);
-- as duas rotas sao idempotentes e podem rodar em qualquer ordem.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        EXECUTE format($f$
            CREATE TABLE IF NOT EXISTS %I.saldo_por_contas (
                id                     BIGSERIAL   PRIMARY KEY,
                empresa_id             UUID        NOT NULL,
                codigo_conta_corrente  BIGINT      NOT NULL,
                data_referencia        DATE        NOT NULL,
                saldo_anterior         NUMERIC(15,2),
                saldo_atual            NUMERIC(15,2),
                saldo_atual_previsto   NUMERIC(15,2),
                raw                    JSONB,
                synced_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                UNIQUE(empresa_id, codigo_conta_corrente, data_referencia)
            )
        $f$, r.schema_name);

        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_saldo_por_contas_data ON %I.saldo_por_contas (empresa_id, data_referencia)', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_saldo_por_contas_cc ON %I.saldo_por_contas (empresa_id, codigo_conta_corrente)', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_saldo_por_contas_raw ON %I.saldo_por_contas USING GIN (raw)', r.schema_name, r.schema_name);

    END LOOP;
END
$$;

-- Configuracao do endpoint. modulo precisa bater exatamente com
-- SaldoPorContasExecutor.Nome(). Reusa o mesmo endpoint/action do modulo
-- extrato, mas com payload diferente (cExibirApenasSaldo=S, datas iguais =
-- dia 1 do mes) — por isso precisa de uma linha propria, a chave de busca do
-- worker e por nome de executor (modulo), nao por endpoint/action.
-- array_field fica como 'listaMovimentos' so pela constraint NOT NULL: com
-- cExibirApenasSaldo=S esse array vem sempre vazio, os campos reais (nSaldoAtual
-- etc.) sao extraidos soltos do envelope, igual a cFluxoCaixa no modulo extrato.
-- ignorar_delta=true porque a incrementalidade e propria do executor (ver
-- internal/etl/saldo_por_contas.go periodosParaSincronizar), nao o buildPaginacao padrao.
INSERT INTO _etl.omie_endpoint_config
    (modulo, endpoint_path, action, array_field, page_size, ignorar_delta, notas)
VALUES
    ('saldo_por_contas', '/financas/extrato/', 'ListarExtrato', 'listaMovimentos', 0, true,
     'Saldo real por conta corrente no dia 1 de cada mes (cExibirApenasSaldo=S). listaMovimentos vem sempre vazia nesse payload; os campos reais vem soltos no envelope (nSaldoAtual, nSaldoAnterior, nSaldoAtualPrevisto, cFluxoCaixa). Backfill: ano corrente + ano anterior. Ultimos 3 meses sao re-sincronizados a cada ciclo (conciliacao bancaria tardia do Omie); meses mais antigos, uma vez gravados, sao tratados como imutaveis.')
ON CONFLICT (modulo) DO NOTHING;
