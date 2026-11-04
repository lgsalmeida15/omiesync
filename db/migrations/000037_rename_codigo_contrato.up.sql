-- Migration 000037: renomeia contratos.codigo_lancamento_omie -> codigo_contrato_omie.
--
-- A coluna sempre guardou o nCodCtr do cabecalho. O nome antigo sugeria que ela era
-- a chave de ligacao com o extrato, o que nao e verdade: a validacao em producao
-- mostrou que a ligacao e numero_contrato (cNumCtr) contra extrato.raw ->> 'cNumero'.
-- codigo_contrato_omie e a chave de identidade da tabela, nao a de ligacao.
--
-- Guardado por IF EXISTS porque schemas provisionados depois do bump para a versao 12
-- ja nascem com o nome novo. A constraint UNIQUE acompanha a coluna automaticamente.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'contratos'
              AND column_name  = 'codigo_lancamento_omie'
        ) THEN
            EXECUTE format('ALTER TABLE %I.contratos RENAME COLUMN codigo_lancamento_omie TO codigo_contrato_omie', r.schema_name);
        END IF;

    END LOOP;
END
$$;

UPDATE _etl.omie_endpoint_config
   SET notas = 'Contratos de servico. apenas_importado_api=N fixo no executor. Campos principais vem aninhados em "cabecalho". Identidade: cabecalho.nCodCtr em codigo_contrato_omie. Ligacao com o extrato: numero_contrato (cNumCtr) <-> extrato.raw->>cNumero.'
 WHERE modulo = 'contratos';
