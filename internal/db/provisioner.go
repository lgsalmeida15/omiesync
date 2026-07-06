package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Provisioner cria e inicializa schemas de tenant.
type Provisioner struct {
	pool *pgxpool.Pool
}

func NewProvisioner(pool *pgxpool.Pool) *Provisioner {
	return &Provisioner{pool: pool}
}

// ProvisionSchema cria o schema do tenant e todas as tabelas Omie necessarias.
// E idempotente — pode ser chamado multiplas vezes sem efeitos colaterais.
func (p *Provisioner) ProvisionSchema(ctx context.Context, schemaName string) error {
	// Sanitiza o nome do schema para evitar SQL injection
	safe := pgx.Identifier{schemaName}.Sanitize()

	stmts := []string{
		// Schema
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", safe),

		// Clientes
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.clientes (
			id                  BIGSERIAL   PRIMARY KEY,
			empresa_id          UUID        NOT NULL,
			codigo_cliente_omie BIGINT      NOT NULL,
			razao_social        TEXT        NOT NULL,
			nome_fantasia       TEXT,
			cnpj_cpf            TEXT,
			email               TEXT,
			telefone1_ddd       TEXT,
			telefone1_numero    TEXT,
			endereco            TEXT,
			cidade              TEXT,
			estado              TEXT,
			cep                 TEXT,
			ativo               BOOLEAN     NOT NULL DEFAULT true,
			data_alteracao      TEXT,
			raw                 JSONB,
			synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_cliente_omie)
		)`, safe),

		// Categorias
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.categorias (
			id                BIGSERIAL   PRIMARY KEY,
			empresa_id        UUID        NOT NULL,
			codigo            TEXT        NOT NULL,
			descricao         TEXT        NOT NULL,
			id_conta_corrente BIGINT,
			raw               JSONB,
			synced_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo)
		)`, safe),

		// Departamentos
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.departamentos (
			id         BIGSERIAL   PRIMARY KEY,
			empresa_id UUID        NOT NULL,
			codigo     TEXT        NOT NULL,
			descricao  TEXT        NOT NULL,
			inativo    TEXT,
			raw        JSONB,
			synced_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo)
		)`, safe),

		// Contas Correntes
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.contas_correntes (
			id                    BIGSERIAL   PRIMARY KEY,
			empresa_id            UUID        NOT NULL,
			codigo_conta_corrente BIGINT      NOT NULL,
			descricao             TEXT        NOT NULL,
			tipo                  TEXT,
			saldo_inicial         NUMERIC(15,2),
			raw                   JSONB,
			synced_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_conta_corrente)
		)`, safe),

		// Contas a Pagar
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.contas_pagar (
			id                BIGSERIAL   PRIMARY KEY,
			empresa_id        UUID        NOT NULL,
			codigo_lancamento BIGINT      NOT NULL,
			data_vencimento   DATE,
			data_previsao     DATE,
			data_pagamento    DATE,
			valor_documento   NUMERIC(15,2),
			valor_pago        NUMERIC(15,2),
			status_titulo     TEXT,
			codigo_cliente    BIGINT,
			codigo_categoria  TEXT,
			observacao        TEXT,
			raw               JSONB,
			synced_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_lancamento)
		)`, safe),

		// Contas a Receber
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.contas_receber (
			id                BIGSERIAL   PRIMARY KEY,
			empresa_id        UUID        NOT NULL,
			codigo_lancamento BIGINT      NOT NULL,
			data_vencimento   DATE,
			data_previsao     DATE,
			data_recebimento  DATE,
			valor_documento   NUMERIC(15,2),
			valor_recebido    NUMERIC(15,2),
			status_titulo     TEXT,
			codigo_cliente    BIGINT,
			codigo_categoria  TEXT,
			observacao        TEXT,
			raw               JSONB,
			synced_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_lancamento)
		)`, safe),

		// Movimentos Financeiros
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.movimentos_financeiros (
			id                    BIGSERIAL   PRIMARY KEY,
			empresa_id            UUID        NOT NULL,
			codigo_lancamento     BIGINT      NOT NULL,
			data_lancamento       DATE,
			valor                 NUMERIC(15,2),
			tipo                  TEXT,
			codigo_conta_corrente BIGINT,
			codigo_categoria      TEXT,
			historico             TEXT,
			raw                   JSONB,
			synced_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_lancamento)
		)`, safe),

		// Extrato — sem UNIQUE pois Omie nao retorna ID por movimento; isolado por empresa_id
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.extrato (
			id                    BIGSERIAL   PRIMARY KEY,
			empresa_id            UUID        NOT NULL,
			codigo_lancamento     BIGINT,
			data_lancamento       DATE,
			valor                 NUMERIC(15,2),
			tipo_lancamento       TEXT,
			codigo_conta_corrente BIGINT,
			descricao             TEXT,
			raw                   JSONB,
			synced_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, safe),

		// Ordens de Servico
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.ordens_servico (
			id              BIGSERIAL   PRIMARY KEY,
			empresa_id      UUID        NOT NULL,
			numero_os       BIGINT      NOT NULL,
			data_abertura   DATE,
			data_previsao   DATE,
			data_fechamento DATE,
			status          TEXT,
			codigo_cliente  BIGINT,
			valor_total     NUMERIC(15,2),
			descricao       TEXT,
			raw             JSONB,
			synced_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, numero_os)
		)`, safe),

		// Projetos
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.projetos (
			id             BIGSERIAL   PRIMARY KEY,
			empresa_id     UUID        NOT NULL,
			codigo_projeto BIGINT      NOT NULL,
			nome           TEXT        NOT NULL,
			descricao      TEXT,
			data_inicio    DATE,
			data_fim       DATE,
			status         TEXT,
			raw            JSONB,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(empresa_id, codigo_projeto)
		)`, safe),

		// Indices de performance
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_clientes_cnpj ON %s.clientes (empresa_id, cnpj_cpf)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cp_venc ON %s.contas_pagar (empresa_id, data_vencimento, status_titulo)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cr_venc ON %s.contas_receber (empresa_id, data_vencimento, status_titulo)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mov_data ON %s.movimentos_financeiros (empresa_id, data_lancamento)", schemaName, safe),
	}

	for _, stmt := range stmts {
		if _, err := p.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("db.Provisioner.ProvisionSchema [%s]: %w", schemaName, err)
		}
	}

	return nil
}

// DropSchema remove completamente o schema de um tenant (chamado pelo deletion job).
func (p *Provisioner) DropSchema(ctx context.Context, schemaName string) error {
	safe := pgx.Identifier{schemaName}.Sanitize()
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", safe))
	if err != nil {
		return fmt.Errorf("db.Provisioner.DropSchema [%s]: %w", schemaName, err)
	}
	return nil
}
