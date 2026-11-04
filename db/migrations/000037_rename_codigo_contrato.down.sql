-- Reverte a migration 000037.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'contratos'
              AND column_name  = 'codigo_contrato_omie'
        ) THEN
            EXECUTE format('ALTER TABLE %I.contratos RENAME COLUMN codigo_contrato_omie TO codigo_lancamento_omie', r.schema_name);
        END IF;

    END LOOP;
END
$$;
