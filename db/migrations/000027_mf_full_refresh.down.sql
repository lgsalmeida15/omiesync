-- Reverte a 000027.
--
-- ATENCAO: a volta da constraint UNIQUE(empresa_id, codigo_lancamento) falha se a tabela
-- ja tiver sido recarregada pelo novo ETL, porque passam a existir multiplas linhas por
-- titulo (baixa em lote) e linhas sem titulo. Nesse caso e preciso deduplicar antes,
-- ou truncar a tabela e rodar um sync com a versao antiga do codigo.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        EXECUTE format('DROP INDEX IF EXISTS %I.idx_%s_mf_titulo', r.schema_name, r.schema_name);
        EXECUTE format('DROP INDEX IF EXISTS %I.idx_%s_mf_mov_cc', r.schema_name, r.schema_name);

        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP COLUMN IF EXISTS codigo_mov_cc', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP COLUMN IF EXISTS codigo_mov_cc_repet', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP COLUMN IF EXISTS codigo_titrepet', r.schema_name);

        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'movimentos_financeiros'
              AND column_name  = 'codigo_titulo'
        ) THEN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros RENAME COLUMN codigo_titulo TO codigo_lancamento', r.schema_name);
        END IF;

        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'movimentos_financeiros'
              AND column_name  = 'numero_titulo'
        ) THEN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros RENAME COLUMN numero_titulo TO historico', r.schema_name);
        END IF;

        -- Linhas sem titulo nao cabem no modelo antigo (coluna era NOT NULL).
        EXECUTE format('DELETE FROM %I.movimentos_financeiros WHERE codigo_lancamento IS NULL', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ALTER COLUMN codigo_lancamento SET NOT NULL', r.schema_name);

        BEGIN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD CONSTRAINT movimentos_financeiros_empresa_codigo_key UNIQUE(empresa_id, codigo_lancamento)', r.schema_name);
        EXCEPTION WHEN duplicate_table OR unique_violation THEN NULL;
        END;

        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_mf_lancamento ON %I.movimentos_financeiros (empresa_id, codigo_lancamento)', r.schema_name, r.schema_name);

    END LOOP;
END
$$;

UPDATE _etl.omie_endpoint_config
SET page_size = 50,
    notas     = 'Paginação especial: nPagina/nRegPorPagina; sempre busca tudo',
    updated_at = NOW()
WHERE modulo = 'movimentos_financeiros';
