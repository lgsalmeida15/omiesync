-- Migration 000021: adiciona empresa_id a todas as tabelas de tenant existentes.
-- Necessaria para isolamento multi-empresa dentro de schemas de grupo compartilhados.
-- Aplica dinamicamente em todos os schemas de grupo ativos.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT schema_name FROM _etl.grupos WHERE deleted_at IS NULL LOOP

        -- clientes
        EXECUTE format('ALTER TABLE %I.clientes ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.clientes DROP CONSTRAINT IF EXISTS clientes_codigo_cliente_omie_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.clientes DROP CONSTRAINT IF EXISTS clientes_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.clientes ADD CONSTRAINT clientes_empresa_codigo_key UNIQUE(empresa_id, codigo_cliente_omie)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- categorias
        EXECUTE format('ALTER TABLE %I.categorias ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.categorias DROP CONSTRAINT IF EXISTS categorias_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.categorias DROP CONSTRAINT IF EXISTS categorias_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.categorias ADD CONSTRAINT categorias_empresa_codigo_key UNIQUE(empresa_id, codigo)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- departamentos
        EXECUTE format('ALTER TABLE %I.departamentos ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.departamentos DROP CONSTRAINT IF EXISTS departamentos_codigo_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.departamentos DROP CONSTRAINT IF EXISTS departamentos_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.departamentos ADD CONSTRAINT departamentos_empresa_codigo_key UNIQUE(empresa_id, codigo)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- contas_correntes
        EXECUTE format('ALTER TABLE %I.contas_correntes ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_correntes DROP CONSTRAINT IF EXISTS contas_correntes_codigo_conta_corrente_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_correntes DROP CONSTRAINT IF EXISTS contas_correntes_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.contas_correntes ADD CONSTRAINT contas_correntes_empresa_codigo_key UNIQUE(empresa_id, codigo_conta_corrente)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- contas_pagar
        EXECUTE format('ALTER TABLE %I.contas_pagar ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_pagar DROP CONSTRAINT IF EXISTS contas_pagar_codigo_lancamento_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_pagar DROP CONSTRAINT IF EXISTS contas_pagar_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.contas_pagar ADD CONSTRAINT contas_pagar_empresa_codigo_key UNIQUE(empresa_id, codigo_lancamento)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- contas_receber
        EXECUTE format('ALTER TABLE %I.contas_receber ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_receber DROP CONSTRAINT IF EXISTS contas_receber_codigo_lancamento_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.contas_receber DROP CONSTRAINT IF EXISTS contas_receber_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.contas_receber ADD CONSTRAINT contas_receber_empresa_codigo_key UNIQUE(empresa_id, codigo_lancamento)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- movimentos_financeiros
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_codigo_lancamento_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.movimentos_financeiros ADD CONSTRAINT movimentos_financeiros_empresa_codigo_key UNIQUE(empresa_id, codigo_lancamento)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- extrato (sem UNIQUE — apenas adicionar coluna)
        EXECUTE format('ALTER TABLE %I.extrato ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);

        -- ordens_servico
        EXECUTE format('ALTER TABLE %I.ordens_servico ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.ordens_servico DROP CONSTRAINT IF EXISTS ordens_servico_numero_os_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.ordens_servico DROP CONSTRAINT IF EXISTS ordens_servico_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.ordens_servico ADD CONSTRAINT ordens_servico_empresa_codigo_key UNIQUE(empresa_id, numero_os)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

        -- projetos
        EXECUTE format('ALTER TABLE %I.projetos ADD COLUMN IF NOT EXISTS empresa_id UUID', r.schema_name);
        EXECUTE format('ALTER TABLE %I.projetos DROP CONSTRAINT IF EXISTS projetos_codigo_projeto_key', r.schema_name);
        EXECUTE format('ALTER TABLE %I.projetos DROP CONSTRAINT IF EXISTS projetos_empresa_codigo_key', r.schema_name);
        BEGIN
            EXECUTE format('ALTER TABLE %I.projetos ADD CONSTRAINT projetos_empresa_codigo_key UNIQUE(empresa_id, codigo_projeto)', r.schema_name);
        EXCEPTION WHEN duplicate_table THEN NULL;
        END;

    END LOOP;
END $$;
