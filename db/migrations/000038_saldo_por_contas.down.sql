-- Reverte a migration 000038.

DELETE FROM _etl.omie_endpoint_config WHERE modulo = 'saldo_por_contas';

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I.saldo_por_contas', r.schema_name);
    END LOOP;
END
$$;
