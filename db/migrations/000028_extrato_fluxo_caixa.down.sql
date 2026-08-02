-- Reverte a 000028. O ramo ext da matvw volta a nao retornar nada e o card
-- SALDO CONTAS volta a zero, porque cFluxoCaixa nao existe em contas_correntes.raw.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I.idx_%s_extrato_fluxo', r.schema_name, r.schema_name);
        EXECUTE format('ALTER TABLE %I.extrato           DROP COLUMN IF EXISTS fluxo_caixa', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_correntes  DROP COLUMN IF EXISTS fluxo_caixa', r.schema_name);
    END LOOP;
END
$$;
