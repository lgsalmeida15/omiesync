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

		// Índices para a view gerencial — extrato
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_data ON %s.extrato (empresa_id, data_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_cc ON %s.extrato (empresa_id, codigo_conta_corrente)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_raw ON %s.extrato USING GIN (raw)", schemaName, safe),

		// Índices para a view gerencial — movimentos_financeiros
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mf_lancamento ON %s.movimentos_financeiros (empresa_id, codigo_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mf_raw ON %s.movimentos_financeiros USING GIN (raw)", schemaName, safe),

		// Índices para a view gerencial — contas_pagar / contas_receber
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cp_lancamento ON %s.contas_pagar (empresa_id, codigo_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cp_raw ON %s.contas_pagar USING GIN (raw)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cr_lancamento ON %s.contas_receber (empresa_id, codigo_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cr_raw ON %s.contas_receber USING GIN (raw)", schemaName, safe),

		// Materialized view gerencial — resultado financeiro do ano corrente
		// WITH NO DATA: populada posteriormente via REFRESH MATERIALIZED VIEW.
		// REFRESH sem CONCURRENTLY (sem necessidade de UNIQUE index em colunas reais).
		fmt.Sprintf(`CREATE MATERIALIZED VIEW IF NOT EXISTS %s.matvw_gerencial_resultado AS
WITH categorias_processadas AS (
    SELECT codigo, descricao FROM %s.categorias
),
movimentos_unificados AS (
    -- Lado extrato: provisões futuras de contas com fluxo de caixa
    SELECT
        e.codigo_conta_corrente::TEXT          AS codigo_conta_corrente,
        e.raw ->> 'nCodCliente'                AS codigo_cliente,
        e.raw ->> 'nCodLancamento'             AS codigo_titulo,
        NULL::TEXT                             AS grupo,
        TO_CHAR(e.data_lancamento, 'DD/MM/YYYY') AS data_pagamento,
        NULL::TEXT                             AS data_previsao,
        e.raw ->> 'cCodCategoria'              AS codigo_categoria,
        NULL::TEXT                             AS status_mov,
        ABS(e.valor)                           AS valor_titulo_mov_ext,
        EXTRACT(YEAR  FROM e.data_lancamento)::INT AS ano,
        EXTRACT(MONTH FROM e.data_lancamento)::INT AS mes,
        'ext'::TEXT                            AS mov_ou_extrato
    FROM %s.extrato e
    JOIN %s.contas_correntes cc
        ON cc.codigo_conta_corrente = e.codigo_conta_corrente
       AND cc.empresa_id = e.empresa_id
    WHERE EXTRACT(YEAR FROM e.data_lancamento) = EXTRACT(YEAR FROM CURRENT_DATE)
      AND cc.raw ->> 'cFluxoCaixa' = 'S'
      AND e.raw ->> 'cSituacao'    = 'Previsto'

    UNION ALL

    -- Lado movimentos: lançamentos realizados (CONTA_CORRENTE_REC / CONTA_CORRENTE_PAG)
    SELECT
        mf.codigo_conta_corrente::TEXT                            AS codigo_conta_corrente,
        mf.raw -> 'detalhes' ->> 'nCodCliente'                   AS codigo_cliente,
        mf.codigo_lancamento::TEXT                                AS codigo_titulo,
        mf.raw -> 'detalhes' ->> 'cGrupo'                        AS grupo,
        mf.raw -> 'detalhes' ->> 'dDtPagamento'                  AS data_pagamento,
        mf.raw -> 'detalhes' ->> 'dDtPrevisao'                   AS data_previsao,
        mf.codigo_categoria                                       AS codigo_categoria,
        mf.raw -> 'detalhes' ->> 'cStatus'                       AS status_mov,
        COALESCE(
            NULLIF(mf.raw -> 'resumo' ->> 'nValLiquido', '')::NUMERIC,
            NULLIF(mf.raw -> 'resumo' ->> 'nValPago',    '')::NUMERIC
        )                                                         AS valor_titulo_mov_ext,
        EXTRACT(YEAR  FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS ano,
        EXTRACT(MONTH FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS mes,
        'mov'::TEXT                                               AS mov_ou_extrato
    FROM %s.movimentos_financeiros mf
    WHERE mf.raw -> 'detalhes' ->> 'cGrupo' IN ('CONTA_CORRENTE_REC','CONTA_CORRENTE_PAG')
      AND NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento','') IS NOT NULL
      AND EXTRACT(YEAR FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))
          = EXTRACT(YEAR FROM CURRENT_DATE)
),
-- Expande categorias de contas_pagar e contas_receber (array raw -> 'categorias')
cp_categorias AS (
    SELECT
        cp.codigo_lancamento::TEXT                      AS id,
        cp.valor_documento,
        (cat_elem ->> 'valor')::NUMERIC                AS valor_categoria,
        (cat_elem ->> 'percentual')::NUMERIC           AS percentual_categoria,
        cat_elem ->> 'codigo_categoria'                AS codigo_categoria,
        'contas_a_pagar'::TEXT                         AS origem
    FROM %s.contas_pagar cp,
         LATERAL jsonb_array_elements(cp.raw -> 'categorias') AS cat_elem
    UNION ALL
    SELECT
        cr.codigo_lancamento::TEXT,
        cr.valor_documento,
        (cat_elem ->> 'valor')::NUMERIC,
        (cat_elem ->> 'percentual')::NUMERIC,
        cat_elem ->> 'codigo_categoria',
        'contas_a_receber'::TEXT
    FROM %s.contas_receber cr,
         LATERAL jsonb_array_elements(cr.raw -> 'categorias') AS cat_elem
),
-- Expande departamentos de contas_pagar e contas_receber (array raw -> 'distribuicao')
cp_distribuicao AS (
    SELECT
        cp.codigo_lancamento::TEXT                     AS id,
        (dist_elem ->> 'nValDep')::NUMERIC             AS valor_distribuido,
        (dist_elem ->> 'nPerDep')::NUMERIC             AS percentual_distribuicao,
        dist_elem ->> 'cCodDep'                        AS codigo_departamento
    FROM %s.contas_pagar cp,
         LATERAL jsonb_array_elements(cp.raw -> 'distribuicao') AS dist_elem
    UNION ALL
    SELECT
        cr.codigo_lancamento::TEXT,
        (dist_elem ->> 'nValDep')::NUMERIC,
        (dist_elem ->> 'nPerDep')::NUMERIC,
        dist_elem ->> 'cCodDep'
    FROM %s.contas_receber cr,
         LATERAL jsonb_array_elements(cr.raw -> 'distribuicao') AS dist_elem
),
movimentos_processados AS (
    SELECT
        m.*,
        c.codigo_categoria                             AS categoria_join,
        c.percentual_categoria,
        c.origem,
        d.codigo_departamento                          AS codigo_departamento_join,
        d.percentual_distribuicao                      AS percentual_distribuicao_join,
        d.valor_distribuido                            AS valor_distribuido_join,
        cc.raw ->> 'cFluxoCaixa'                      AS conta_considerada,
        LEFT(COALESCE(c.codigo_categoria, m.codigo_categoria), 4) AS cod_categoria_final_superior,
        cat_sup.descricao                              AS descricao_categoria_superior,
        COALESCE(c.codigo_categoria, m.codigo_categoria) AS cod_categoria_final,
        COALESCE(c.percentual_categoria, 100)          AS percentual_cat_final,
        CASE
            WHEN c.origem = 'contas_a_receber'                                              THEN 'receita'
            WHEN c.origem = 'contas_a_pagar'                                                THEN 'despesa'
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'mov' AND UPPER(m.status_mov) = 'RECEBIDO' THEN 'receita'
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'mov' AND UPPER(m.status_mov) = 'PAGO'     THEN 'despesa'
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'ext' AND m.valor_titulo_mov_ext > 0        THEN 'receita'
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'ext' AND m.valor_titulo_mov_ext < 0        THEN 'despesa'
        END                                            AS receita_despesa,
        m.valor_titulo_mov_ext * COALESCE(c.percentual_categoria, 100) / 100.0 AS valor_final,
        CASE
            WHEN m.mov_ou_extrato = 'ext'                                                   THEN 1
            WHEN m.mov_ou_extrato = 'mov'
             AND m.grupo IN ('CONTA_CORRENTE_PAG','CONTA_CORRENTE_REC')                     THEN 1
            ELSE 0
        END                                            AS movimento_considerado,
        cat_final.descricao                            AS descricao_categoria_final,
        CASE
            WHEN d.codigo_departamento IS NULL THEN 'Sem departamento'
            ELSE dept.descricao
        END                                            AS departamento_final,
        COALESCE(cli.nome_fantasia, 'Cliente não informado') AS cliente_final,
        CASE
            WHEN c.origem = 'contas_a_receber'                                              THEN 1
            WHEN c.origem = 'contas_a_pagar'                                                THEN 2
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'mov' AND UPPER(m.status_mov) = 'RECEBIDO' THEN 1
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'mov' AND UPPER(m.status_mov) = 'PAGO'     THEN 2
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'ext' AND m.valor_titulo_mov_ext > 0        THEN 1
            WHEN c.origem IS NULL AND m.mov_ou_extrato = 'ext' AND m.valor_titulo_mov_ext < 0        THEN 2
            ELSE 2
        END                                            AS ajuste_receita_despesa
    FROM movimentos_unificados m
    LEFT JOIN cp_categorias   c    ON m.codigo_titulo = c.id
    LEFT JOIN cp_distribuicao d    ON m.codigo_titulo = d.id
    LEFT JOIN %s.contas_correntes cc
           ON cc.codigo_conta_corrente::TEXT = m.codigo_conta_corrente
    LEFT JOIN categorias_processadas cat_sup
           ON LEFT(COALESCE(c.codigo_categoria, m.codigo_categoria), 4) = cat_sup.codigo
    LEFT JOIN categorias_processadas cat_final
           ON COALESCE(c.codigo_categoria, m.codigo_categoria) = cat_final.codigo
    LEFT JOIN %s.departamentos dept
           ON d.codigo_departamento = dept.codigo
    LEFT JOIN %s.clientes cli
           ON cli.codigo_cliente_omie::TEXT = m.codigo_cliente
)
SELECT
    codigo_conta_corrente,
    codigo_cliente,
    codigo_titulo,
    grupo,
    data_pagamento,
    data_previsao,
    codigo_categoria,
    status_mov,
    valor_titulo_mov_ext,
    ano,
    mes,
    mov_ou_extrato,
    categoria_join,
    percentual_categoria,
    origem,
    codigo_departamento_join,
    percentual_distribuicao_join,
    valor_distribuido_join,
    conta_considerada,
    cod_categoria_final_superior,
    cod_categoria_final,
    percentual_cat_final,
    receita_despesa,
    ajuste_receita_despesa,
    valor_final,
    movimento_considerado,
    descricao_categoria_superior,
    descricao_categoria_final,
    departamento_final,
    cliente_final
FROM movimentos_processados
WHERE movimento_considerado = 1
WITH NO DATA`,
			safe,                   // %s.matvw_gerencial_resultado
			safe,                   // categorias_processadas: %s.categorias
			safe, safe,             // movimentos_unificados extrato: %s.extrato, %s.contas_correntes
			safe,                   // movimentos_unificados mf: %s.movimentos_financeiros
			safe, safe,             // cp_categorias: %s.contas_pagar, %s.contas_receber
			safe, safe,             // cp_distribuicao: %s.contas_pagar, %s.contas_receber
			safe,                   // movimentos_processados: %s.contas_correntes
			safe,                   // movimentos_processados: %s.departamentos
			safe,                   // movimentos_processados: %s.clientes
		),

		// Índices na materialized view (REFRESH sem CONCURRENTLY — não requer UNIQUE)
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mvw_ano_mes ON %s.matvw_gerencial_resultado (ano, mes)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mvw_receita ON %s.matvw_gerencial_resultado (receita_despesa)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mvw_categoria ON %s.matvw_gerencial_resultado (cod_categoria_final)", schemaName, safe),
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
