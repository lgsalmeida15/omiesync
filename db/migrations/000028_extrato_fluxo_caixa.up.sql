-- Migration 000028: persiste o cFluxoCaixa, que hoje e descartado.
--
-- Contexto: no sistema legado o extrato ficava aninhado dentro do registro da conta
-- corrente, e cFluxoCaixa era irmao de listaMovimentos dentro da MESMA resposta do
-- ListarExtrato. Hoje o extrato e tabela propria e o executor le apenas
-- resp["listaMovimentos"] (extrato.go:195) — o envelope da resposta, onde vive o
-- cFluxoCaixa, e jogado fora.
--
-- Consequencia: a matvw procurava o campo em contas_correntes.raw, que vem de outro
-- endpoint (/geral/contacorrente/) e nao possui o campo. Confirmado em producao:
-- 21 contas, ZERO com cFluxoCaixa. Como o JOIN era INNER, o ramo ext da view retornava
-- zero linhas sempre. O mesmo filtro zerava o card SALDO CONTAS do dashboard.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        -- Por linha de extrato: o valor vem do envelope da resposta daquela conta/periodo.
        EXECUTE format('ALTER TABLE %I.extrato ADD COLUMN IF NOT EXISTS fluxo_caixa TEXT', r.schema_name);

        -- Por conta corrente: alimentada pelo executor de extrato, unica fonte do campo.
        -- Necessaria para o card de saldo, que precisa do flag mesmo em conta sem extrato.
        EXECUTE format('ALTER TABLE %I.contas_correntes ADD COLUMN IF NOT EXISTS fluxo_caixa TEXT', r.schema_name);

        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_extrato_fluxo ON %I.extrato (empresa_id, fluxo_caixa)', r.schema_name, r.schema_name);

    END LOOP;
END
$$;
