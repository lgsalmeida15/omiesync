-- Migration 000027: movimentos_financeiros passa de upsert por titulo para carga completa.
--
-- Contexto: a chave UNIQUE(empresa_id, codigo_lancamento) usava nCodTitulo, que NAO e
-- unico por movimento — uma baixa em lote liquida N titulos com um unico nCodMovCC, e
-- movimentos sem titulo (tarifas, transferencias) chegam com nCodTitulo = 0 e eram
-- descartados silenciosamente pelo ETL.
--
-- A partir daqui a tabela nao tem chave de negocio: cada sync apaga e regrava os dados
-- da empresa dentro de uma transacao. A corretude depende dessa transacao, nao de constraint.
--
-- Renomeacoes corrigem nomes que mentiam sobre o conteudo:
--   codigo_lancamento -> codigo_titulo   (sempre guardou nCodTitulo)
--   historico         -> numero_titulo   (sempre guardou cNumTitulo)

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        -- 1. Remove a chave de negocio. Dois nomes possiveis conforme a epoca em que o
        --    schema foi criado: auto-nomeada pelo UNIQUE inline do provisioner, ou
        --    explicita pela migration 000021.
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_id_codigo_lancamento_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_codigo_lancamento_key', r.schema_name);

        -- 2. Renomeia as colunas cujo nome nao correspondia ao conteudo.
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'movimentos_financeiros'
              AND column_name  = 'codigo_lancamento'
        ) THEN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros RENAME COLUMN codigo_lancamento TO codigo_titulo', r.schema_name);
        END IF;

        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'movimentos_financeiros'
              AND column_name  = 'historico'
        ) THEN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros RENAME COLUMN historico TO numero_titulo', r.schema_name);
        END IF;

        -- 3. Movimento sem titulo precisa poder ser gravado.
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ALTER COLUMN codigo_titulo DROP NOT NULL', r.schema_name);

        -- 4. Campos que existiam no payload e nao eram persistidos.
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_mov_cc       BIGINT', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_mov_cc_repet BIGINT', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_titrepet     BIGINT', r.schema_name);

        -- 5. Backfill a partir do raw — nao exige rede nem re-sync.
        EXECUTE format($f$
            UPDATE %I.movimentos_financeiros
            SET codigo_mov_cc       = NULLIF(raw->'detalhes'->>'nCodMovCC','')::BIGINT,
                codigo_mov_cc_repet = NULLIF(raw->'detalhes'->>'nCodMovCCRepet','')::BIGINT,
                codigo_titrepet     = NULLIF(raw->'detalhes'->>'nCodTitRepet','')::BIGINT
            WHERE codigo_mov_cc IS NULL
        $f$, r.schema_name);

        -- 6. nCodTitulo = 0 significa "sem titulo" — normaliza para NULL.
        EXECUTE format('UPDATE %I.movimentos_financeiros SET codigo_titulo = NULL WHERE codigo_titulo = 0', r.schema_name);

        -- 7. Indices. O de codigo_titulo deixa de ser unico mas continua sendo a chave
        --    do join da matvw com contas_pagar/contas_receber.
        EXECUTE format('DROP INDEX IF EXISTS %I.idx_%s_mf_lancamento', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_mf_titulo  ON %I.movimentos_financeiros (empresa_id, codigo_titulo)', r.schema_name, r.schema_name);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_mf_mov_cc  ON %I.movimentos_financeiros (empresa_id, codigo_mov_cc)', r.schema_name, r.schema_name);

    END LOOP;
END
$$;

-- /financas/mf/ e o unico endpoint Omie que aceita 500 registros por pagina.
-- 21k registros passam de ~430 para ~43 chamadas por sync.
UPDATE _etl.omie_endpoint_config
SET page_size = 500,
    notas     = 'Paginacao especial: nPagina/nRegPorPagina; sempre busca tudo. Unico endpoint que aceita 500/pagina.',
    updated_at = NOW()
WHERE modulo = 'movimentos_financeiros';
