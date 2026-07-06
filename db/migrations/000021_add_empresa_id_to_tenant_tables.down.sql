-- Rollback 000021: remove empresa_id e restaura constraints únicas simples em todos os tenants.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        EXECUTE format('ALTER TABLE %I.clientes DROP CONSTRAINT IF EXISTS clientes_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.clientes DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.clientes ADD CONSTRAINT clientes_codigo_cliente_omie_key UNIQUE(codigo_cliente_omie)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.categorias DROP CONSTRAINT IF EXISTS categorias_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.categorias DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.categorias ADD CONSTRAINT categorias_codigo_key UNIQUE(codigo)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.departamentos DROP CONSTRAINT IF EXISTS departamentos_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.departamentos DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.departamentos ADD CONSTRAINT departamentos_codigo_key UNIQUE(codigo)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.contas_correntes DROP CONSTRAINT IF EXISTS contas_correntes_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_correntes DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_correntes ADD CONSTRAINT contas_correntes_codigo_conta_corrente_key UNIQUE(codigo_conta_corrente)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.contas_pagar DROP CONSTRAINT IF EXISTS contas_pagar_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_pagar DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_pagar ADD CONSTRAINT contas_pagar_codigo_lancamento_key UNIQUE(codigo_lancamento)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.contas_receber DROP CONSTRAINT IF EXISTS contas_receber_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_receber DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_receber ADD CONSTRAINT contas_receber_codigo_lancamento_key UNIQUE(codigo_lancamento)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD CONSTRAINT movimentos_financeiros_codigo_lancamento_key UNIQUE(codigo_lancamento)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.extrato DROP COLUMN IF EXISTS empresa_id', r.schema_name);

        EXECUTE format('ALTER TABLE %I.ordens_servico DROP CONSTRAINT IF EXISTS ordens_servico_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.ordens_servico DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.ordens_servico ADD CONSTRAINT ordens_servico_numero_os_key UNIQUE(numero_os)', r.schema_name);

        EXECUTE format('ALTER TABLE %I.projetos DROP CONSTRAINT IF EXISTS projetos_empresa_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.projetos DROP COLUMN IF EXISTS empresa_id', r.schema_name);
        EXECUTE format('ALTER TABLE %I.projetos ADD CONSTRAINT projetos_codigo_projeto_key UNIQUE(codigo_projeto)', r.schema_name);

    END LOOP;
END $$;
