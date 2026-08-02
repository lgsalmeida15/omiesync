package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CurrentSchemaVersion é incrementado sempre que ProvisionSchema ganhar novas
// tabelas, views ou índices. O worker só re-provisiona quando a versão gravada
// em _etl.schema_versions for menor que este valor.
const CurrentSchemaVersion = 5

// Provisioner cria e inicializa schemas de tenant.
type Provisioner struct {
	pool *pgxpool.Pool
}

func NewProvisioner(pool *pgxpool.Pool) *Provisioner {
	return &Provisioner{pool: pool}
}

// NeedsProvisioning retorna true se o schema ainda não foi provisionado na
// versão atual (ou se nunca foi provisionado).
func (p *Provisioner) NeedsProvisioning(ctx context.Context, schemaName string) bool {
	var version int
	err := p.pool.QueryRow(ctx,
		`SELECT version FROM _etl.schema_versions WHERE schema_name = $1`,
		schemaName).Scan(&version)
	if err != nil {
		return true // sem entrada = precisa provisionar
	}
	return version < CurrentSchemaVersion
}

// markProvisioned grava (ou atualiza) a versão atual do schema na tabela de controle.
func (p *Provisioner) markProvisioned(ctx context.Context, schemaName string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO _etl.schema_versions (schema_name, version, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (schema_name) DO UPDATE SET version = $2, updated_at = NOW()
	`, schemaName, CurrentSchemaVersion)
	return err
}

// ProvisionSchema cria o schema do tenant e todas as tabelas Omie necessarias.
// É idempotente — pode ser chamado multiplas vezes sem efeitos colaterais.
// Ao concluir com sucesso, marca a versão em _etl.schema_versions.
func (p *Provisioner) ProvisionSchema(ctx context.Context, schemaName string) error {
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
			-- Alimentada pelo executor de extrato: o cadastro de /geral/contacorrente/
			-- não traz cFluxoCaixa, só a resposta do ListarExtrato.
			fluxo_caixa           TEXT,
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
		// Sem chave de negócio: cada sync apaga e regrava os dados da empresa numa
		// transação. nCodTitulo não é único por movimento (baixa em lote liquida N
		// títulos num só nCodMovCC) e movimentos sem título chegam com nCodTitulo = 0.
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.movimentos_financeiros (
			id                    BIGSERIAL   PRIMARY KEY,
			empresa_id            UUID        NOT NULL,
			codigo_titulo         BIGINT,
			numero_titulo         TEXT,
			codigo_mov_cc         BIGINT,
			codigo_mov_cc_repet   BIGINT,
			codigo_titrepet       BIGINT,
			data_lancamento       DATE,
			valor                 NUMERIC(15,2),
			tipo                  TEXT,
			codigo_conta_corrente BIGINT,
			codigo_categoria      TEXT,
			raw                   JSONB,
			synced_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
			-- cFluxoCaixa vem do envelope da resposta do ListarExtrato, não do
			-- cadastro da conta corrente. É a única fonte do campo.
			fluxo_caixa           TEXT,
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

		// Índices para as views gerenciais — extrato
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_data ON %s.extrato (empresa_id, data_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_cc ON %s.extrato (empresa_id, codigo_conta_corrente)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_extrato_raw ON %s.extrato USING GIN (raw)", schemaName, safe),

		// Índices para as views gerenciais — movimentos_financeiros
		// codigo_titulo não é único, mas é a chave do join com contas_pagar/contas_receber.
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mf_titulo ON %s.movimentos_financeiros (empresa_id, codigo_titulo)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mf_mov_cc ON %s.movimentos_financeiros (empresa_id, codigo_mov_cc)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_mf_raw ON %s.movimentos_financeiros USING GIN (raw)", schemaName, safe),

		// Índices para as views gerenciais — contas_pagar / contas_receber
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cp_lancamento ON %s.contas_pagar (empresa_id, codigo_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cp_raw ON %s.contas_pagar USING GIN (raw)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cr_lancamento ON %s.contas_receber (empresa_id, codigo_lancamento)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_cr_raw ON %s.contas_receber USING GIN (raw)", schemaName, safe),

		// Auto-upgrade v5: remove a view unificada antiga (matvw_gerencial_resultado),
		// substituída pelas views separadas ano_corrente e historico.
		// Engloba os upgrade blocks v3 e v4 — drop incondicional se existir.
		fmt.Sprintf(`DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class pc
        JOIN pg_namespace pn ON pn.oid = pc.relnamespace
        WHERE pn.nspname = %s AND pc.relname = 'matvw_gerencial_resultado'
          AND pc.relkind = 'm'
    ) THEN
        DROP MATERIALIZED VIEW %s.matvw_gerencial_resultado CASCADE;
    END IF;
END
$$`, "'"+schemaName+"'", safe),

		// Auto-upgrade v6: movimentos_financeiros passa de upsert por título para carga
		// completa. Espelha a migration 000027 para schemas provisionados por binário
		// antigo. Idempotente.
		fmt.Sprintf(`DO $$
BEGIN
    ALTER TABLE %[2]s.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_id_codigo_lancamento_key;
    ALTER TABLE %[2]s.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_empresa_codigo_key;
    ALTER TABLE %[2]s.movimentos_financeiros DROP CONSTRAINT IF EXISTS movimentos_financeiros_codigo_lancamento_key;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = %[1]s AND table_name = 'movimentos_financeiros'
                 AND column_name = 'codigo_lancamento') THEN
        ALTER TABLE %[2]s.movimentos_financeiros RENAME COLUMN codigo_lancamento TO codigo_titulo;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = %[1]s AND table_name = 'movimentos_financeiros'
                 AND column_name = 'historico') THEN
        ALTER TABLE %[2]s.movimentos_financeiros RENAME COLUMN historico TO numero_titulo;
    END IF;

    ALTER TABLE %[2]s.movimentos_financeiros ALTER COLUMN codigo_titulo DROP NOT NULL;
    ALTER TABLE %[2]s.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_mov_cc       BIGINT;
    ALTER TABLE %[2]s.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_mov_cc_repet BIGINT;
    ALTER TABLE %[2]s.movimentos_financeiros ADD COLUMN IF NOT EXISTS codigo_titrepet     BIGINT;
    UPDATE %[2]s.movimentos_financeiros SET codigo_titulo = NULL WHERE codigo_titulo = 0;

    -- As matvw são criadas com IF NOT EXISTS, então uma view já existente manteria a
    -- definição antiga. Dropa apenas se a definição atual for a antiga — detectada pela
    -- ausência do DISTINCT ON introduzido nesta versão. Idempotente: nas execuções
    -- seguintes a condição é falsa e as views são preservadas com seus dados.
    IF EXISTS (SELECT 1 FROM pg_matviews
               WHERE schemaname = %[1]s AND matviewname = 'matvw_gerencial_ano_corrente'
                 AND definition NOT LIKE '%%DISTINCT ON%%') THEN
        DROP MATERIALIZED VIEW %[2]s.matvw_gerencial_ano_corrente CASCADE;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_matviews
               WHERE schemaname = %[1]s AND matviewname = 'matvw_gerencial_historico'
                 AND definition NOT LIKE '%%DISTINCT ON%%') THEN
        DROP MATERIALIZED VIEW %[2]s.matvw_gerencial_historico CASCADE;
    END IF;
END
$$`, "'"+schemaName+"'", safe),

		// Auto-upgrade v7: persiste o cFluxoCaixa, antes descartado. Espelha a migration
		// 000028 para schemas provisionados por binário antigo. Idempotente.
		fmt.Sprintf(`DO $$
BEGIN
    ALTER TABLE %[2]s.extrato          ADD COLUMN IF NOT EXISTS fluxo_caixa TEXT;
    ALTER TABLE %[2]s.contas_correntes ADD COLUMN IF NOT EXISTS fluxo_caixa TEXT;

    -- Recria as views que ainda filtram cFluxoCaixa em contas_correntes.raw, onde o
    -- campo nunca existiu. O marcador é a nova referência a e.fluxo_caixa.
    IF EXISTS (SELECT 1 FROM pg_matviews
               WHERE schemaname = %[1]s AND matviewname = 'matvw_gerencial_ano_corrente'
                 AND definition NOT LIKE '%%fluxo_caixa%%') THEN
        DROP MATERIALIZED VIEW %[2]s.matvw_gerencial_ano_corrente CASCADE;
    END IF;
    -- A histórica perde o ramo ext (extrato só guarda futuro); o marcador é a
    -- ausência da tabela extrato na definição.
    IF EXISTS (SELECT 1 FROM pg_matviews
               WHERE schemaname = %[1]s AND matviewname = 'matvw_gerencial_historico'
                 AND definition LIKE '%%.extrato %%') THEN
        DROP MATERIALIZED VIEW %[2]s.matvw_gerencial_historico CASCADE;
    END IF;
END
$$`, "'"+schemaName+"'", safe),

		// ── View: ano corrente + previsões futuras ────────────────────────────────
		// Filtro: ano >= ano atual (captura previsões de extrato para anos futuros).
		// REFRESH: após todo sync (incremental e full) — dataset pequeno, rápido.
		// WITH NO DATA: populada pelo primeiro REFRESH após criação.
		fmt.Sprintf(`CREATE MATERIALIZED VIEW IF NOT EXISTS %s.matvw_gerencial_ano_corrente AS
WITH categorias_processadas AS (
    SELECT empresa_id, codigo, descricao FROM %s.categorias
),
movimentos_unificados AS (
    -- Extrato: provisões com fluxo de caixa do ano corrente em diante
    SELECT
        e.empresa_id,
        e.codigo_conta_corrente::TEXT          AS codigo_conta_corrente,
        e.raw ->> 'nCodCliente'                AS codigo_cliente,
        e.raw ->> 'nCodLancamento'             AS codigo_titulo,
        NULL::TEXT                             AS grupo,
        TO_CHAR(e.data_lancamento, 'DD/MM/YYYY') AS data_pagamento,
        NULL::TEXT                             AS data_previsao,
        e.raw ->> 'cCodCategoria'              AS codigo_categoria,
        NULL::TEXT                             AS status_mov,
        -- COM SINAL: negativo = saída, positivo = entrada. O ABS() anterior apagava
        -- essa informação e classificava toda provisão como receita.
        e.valor                                AS valor_titulo_mov_ext,
        EXTRACT(YEAR  FROM e.data_lancamento)::INT AS ano,
        EXTRACT(MONTH FROM e.data_lancamento)::INT AS mes,
        'ext'::TEXT                            AS mov_ou_extrato,
        NULL::TEXT                             AS departamento_mov,
        e.raw ->> 'cDesCategoria'              AS descricao_categoria_ext,
        e.raw ->> 'cRazCliente'                AS cliente_ext
    FROM %s.extrato e
    -- LEFT: a conta corrente só enriquece. O filtro de fluxo de caixa vem da própria
    -- linha do extrato, porque cFluxoCaixa é campo da resposta do ListarExtrato e não
    -- existe no cadastro de /geral/contacorrente/.
    LEFT JOIN %s.contas_correntes cc
        ON cc.codigo_conta_corrente = e.codigo_conta_corrente
       AND cc.empresa_id = e.empresa_id
    WHERE e.fluxo_caixa            = 'S'
      AND e.raw ->> 'cSituacao'    = 'Previsto'
      AND e.data_lancamento IS NOT NULL
      AND EXTRACT(YEAR FROM e.data_lancamento) >= EXTRACT(YEAR FROM CURRENT_DATE)

    UNION ALL

    -- Movimentos realizados do ano corrente em diante.
    -- DISTINCT ON mantém o grão de título (uma baixa em lote traz N títulos por
    -- movimento, e o modelo gerencial raciocina por título). Movimentos sem título
    -- usam o próprio id como chave, senão todos colapsariam numa linha só — no
    -- Postgres, NULL agrupa com NULL no DISTINCT ON.
    -- Os parênteses são obrigatórios: sem eles o ORDER BY final ligaria ao UNION
    -- inteiro e o DISTINCT ON seria rejeitado.
    (SELECT DISTINCT ON (mf.empresa_id, COALESCE(mf.codigo_titulo::TEXT, 'mov:' || mf.id::TEXT))
        mf.empresa_id,
        mf.codigo_conta_corrente::TEXT                            AS codigo_conta_corrente,
        mf.raw -> 'detalhes' ->> 'nCodCliente'                   AS codigo_cliente,
        mf.codigo_titulo::TEXT                                    AS codigo_titulo,
        mf.raw -> 'detalhes' ->> 'cGrupo'                        AS grupo,
        mf.raw -> 'detalhes' ->> 'dDtPagamento'                  AS data_pagamento,
        mf.raw -> 'detalhes' ->> 'dDtPrevisao'                   AS data_previsao,
        mf.codigo_categoria                                       AS codigo_categoria,
        mf.raw -> 'detalhes' ->> 'cStatus'                       AS status_mov,
        -- nValLiquido e nValPago cobrem o título; nValorMovCC cobre o lançamento
        -- avulso (tarifa, transferência), que não traz os dois primeiros.
        COALESCE(
            NULLIF(mf.raw -> 'resumo'   ->> 'nValLiquido',  '')::NUMERIC,
            NULLIF(mf.raw -> 'resumo'   ->> 'nValPago',     '')::NUMERIC,
            NULLIF(mf.raw -> 'detalhes' ->> 'nValorMovCC',  '')::NUMERIC
        )                                                         AS valor_titulo_mov_ext,
        EXTRACT(YEAR  FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS ano,
        EXTRACT(MONTH FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS mes,
        'mov'::TEXT                                               AS mov_ou_extrato,
        -- Departamento do próprio movimento: usado quando não há título, já que
        -- nesse caso não existe linha em cp_distribuicao para casar.
        (SELECT dep ->> 'cCodDepartamento'
           FROM jsonb_array_elements(
                    CASE WHEN jsonb_typeof(mf.raw -> 'departamentos') = 'array'
                         THEN mf.raw -> 'departamentos'
                         ELSE '[]'::JSONB END) AS dep
          LIMIT 1)                                                AS departamento_mov,
        NULL::TEXT                                                AS descricao_categoria_ext,
        NULL::TEXT                                                AS cliente_ext
    FROM %s.movimentos_financeiros mf
    WHERE mf.raw -> 'detalhes' ->> 'cGrupo' IN ('CONTA_CORRENTE_REC','CONTA_CORRENTE_PAG')
      AND NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento','') IS NOT NULL
      AND EXTRACT(YEAR FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))
          >= EXTRACT(YEAR FROM CURRENT_DATE)
    ORDER BY mf.empresa_id, COALESCE(mf.codigo_titulo::TEXT, 'mov:' || mf.id::TEXT), mf.id DESC)
),
cp_categorias AS (
    SELECT
        cp.empresa_id,
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
        cr.empresa_id,
        cr.codigo_lancamento::TEXT,
        cr.valor_documento,
        (cat_elem ->> 'valor')::NUMERIC,
        (cat_elem ->> 'percentual')::NUMERIC,
        cat_elem ->> 'codigo_categoria',
        'contas_a_receber'::TEXT
    FROM %s.contas_receber cr,
         LATERAL jsonb_array_elements(cr.raw -> 'categorias') AS cat_elem
),
cp_distribuicao AS (
    SELECT
        cp.empresa_id,
        cp.codigo_lancamento::TEXT                     AS id,
        (dist_elem ->> 'nValDep')::NUMERIC             AS valor_distribuido,
        (dist_elem ->> 'nPerDep')::NUMERIC             AS percentual_distribuicao,
        dist_elem ->> 'cCodDep'                        AS codigo_departamento
    FROM %s.contas_pagar cp,
         LATERAL jsonb_array_elements(cp.raw -> 'distribuicao') AS dist_elem
    UNION ALL
    SELECT
        cr.empresa_id,
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
        COALESCE(d.codigo_departamento, m.departamento_mov) AS codigo_departamento_join,
        d.percentual_distribuicao                      AS percentual_distribuicao_join,
        d.valor_distribuido                            AS valor_distribuido_join,
        cc.fluxo_caixa                                 AS conta_considerada,
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
        -- ABS só no ext, cujo valor agora carrega sinal. O dashboard soma receita e
        -- despesa como grandezas positivas e separa pelo ajuste_receita_despesa.
        -- No mov o valor já é positivo, então o ramo fica idêntico ao anterior.
        CASE WHEN m.mov_ou_extrato = 'ext' THEN ABS(m.valor_titulo_mov_ext)
             ELSE m.valor_titulo_mov_ext END
            * COALESCE(c.percentual_categoria, 100) / 100.0 AS valor_final,
        CASE
            WHEN m.mov_ou_extrato = 'ext'                                                   THEN 1
            WHEN m.mov_ou_extrato = 'mov'
             AND m.grupo IN ('CONTA_CORRENTE_PAG','CONTA_CORRENTE_REC')                     THEN 1
            ELSE 0
        END                                            AS movimento_considerado,
        -- Fronteira temporal: passado vem do realizado (mov), futuro vem do previsto
        -- (ext). Sem isso, um pagamento agendado com data futura no mov somaria em
        -- duplicidade com a provisão correspondente do extrato.
        CASE
            WHEN m.mov_ou_extrato = 'mov'
             AND TO_DATE(NULLIF(m.data_pagamento,''), 'DD/MM/YYYY') <  CURRENT_DATE THEN 1
            WHEN m.mov_ou_extrato = 'ext'
             AND TO_DATE(NULLIF(m.data_pagamento,''), 'DD/MM/YYYY') >= CURRENT_DATE THEN 1
            ELSE 0
        END                                            AS considerar_mov_ext,
        -- O extrato traz cDesCategoria e cRazCliente prontos; servem de fallback
        -- quando a categoria ou o cliente ainda não foram sincronizados.
        COALESCE(cat_final.descricao, m.descricao_categoria_ext) AS descricao_categoria_final,
        -- Título usa a distribuição de contas_pagar/receber; lançamento avulso usa o
        -- array departamentos do próprio movimento.
        CASE
            WHEN COALESCE(d.codigo_departamento, m.departamento_mov) IS NULL THEN 'Sem departamento'
            ELSE dept.descricao
        END                                            AS departamento_final,
        COALESCE(cli.nome_fantasia, m.cliente_ext, 'Cliente não informado') AS cliente_final,
        emp.nome                                       AS nome_empresa,
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
    LEFT JOIN cp_categorias   c    ON m.codigo_titulo = c.id   AND m.empresa_id = c.empresa_id
    LEFT JOIN cp_distribuicao d    ON m.codigo_titulo = d.id   AND m.empresa_id = d.empresa_id
    LEFT JOIN %s.contas_correntes cc
           ON cc.codigo_conta_corrente::TEXT = m.codigo_conta_corrente
          AND cc.empresa_id = m.empresa_id
    LEFT JOIN categorias_processadas cat_sup
           ON LEFT(COALESCE(c.codigo_categoria, m.codigo_categoria), 4) = cat_sup.codigo
          AND cat_sup.empresa_id = m.empresa_id
    LEFT JOIN categorias_processadas cat_final
           ON COALESCE(c.codigo_categoria, m.codigo_categoria) = cat_final.codigo
          AND cat_final.empresa_id = m.empresa_id
    LEFT JOIN %s.departamentos dept
           ON COALESCE(d.codigo_departamento, m.departamento_mov) = dept.codigo
          AND dept.empresa_id = m.empresa_id
    LEFT JOIN %s.clientes cli
           ON cli.codigo_cliente_omie::TEXT = m.codigo_cliente
          AND cli.empresa_id = m.empresa_id
    LEFT JOIN _etl.empresas emp
           ON emp.id = m.empresa_id
)
SELECT
    empresa_id, nome_empresa, codigo_conta_corrente, codigo_cliente,
    codigo_titulo, grupo, data_pagamento, data_previsao, codigo_categoria,
    status_mov, valor_titulo_mov_ext, ano, mes, mov_ou_extrato,
    categoria_join, percentual_categoria, origem, codigo_departamento_join,
    percentual_distribuicao_join, valor_distribuido_join, conta_considerada,
    cod_categoria_final_superior, cod_categoria_final, percentual_cat_final,
    receita_despesa, ajuste_receita_despesa, valor_final,
    movimento_considerado, considerar_mov_ext,
    descricao_categoria_superior, descricao_categoria_final,
    departamento_final, cliente_final
FROM movimentos_processados
WHERE movimento_considerado = 1 AND considerar_mov_ext = 1
WITH NO DATA`,
			safe,           // matvw_gerencial_ano_corrente
			safe,           // categorias_processadas: categorias
			safe, safe,     // extrato, contas_correntes
			safe,           // movimentos_financeiros
			safe, safe,     // cp_categorias: contas_pagar, contas_receber
			safe, safe,     // cp_distribuicao: contas_pagar, contas_receber
			safe,           // movimentos_processados: contas_correntes
			safe,           // movimentos_processados: departamentos
			safe,           // movimentos_processados: clientes
		),

		// Índices: ano_corrente (dataset pequeno — mes é o filtro mais comum)
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_ac_mes ON %s.matvw_gerencial_ano_corrente (mes)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_ac_receita ON %s.matvw_gerencial_ano_corrente (receita_despesa)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_ac_categoria ON %s.matvw_gerencial_ano_corrente (cod_categoria_final)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_ac_empresa ON %s.matvw_gerencial_ano_corrente (empresa_id)", schemaName, safe),

		// ── View: histórico (anos anteriores) ────────────────────────────────────
		// Filtro: ano < ano atual.
		// REFRESH: apenas após sync full — dataset grande, muda raramente.
		// WITH NO DATA: populada pelo REFRESH do primeiro full sync após criação.
		fmt.Sprintf(`CREATE MATERIALIZED VIEW IF NOT EXISTS %s.matvw_gerencial_historico AS
WITH categorias_processadas AS (
    SELECT empresa_id, codigo, descricao FROM %s.categorias
),
movimentos_unificados AS (
    -- Sem ramo de extrato: o executor sincroniza de time.Now() até +1 ano e apaga a
    -- conta antes de gravar (extrato.go:88-89, 96), logo nenhuma linha de extrato pode
    -- ter data passada. Um ramo filtrando ano < ano atual seria sempre vazio e só
    -- custaria tempo de REFRESH.
    --
    -- Movimentos realizados de anos anteriores.
    -- Ver comentário equivalente em matvw_gerencial_ano_corrente sobre o DISTINCT ON
    -- e sobre a obrigatoriedade dos parênteses neste arm.
    (SELECT DISTINCT ON (mf.empresa_id, COALESCE(mf.codigo_titulo::TEXT, 'mov:' || mf.id::TEXT))
        mf.empresa_id,
        mf.codigo_conta_corrente::TEXT                            AS codigo_conta_corrente,
        mf.raw -> 'detalhes' ->> 'nCodCliente'                   AS codigo_cliente,
        mf.codigo_titulo::TEXT                                    AS codigo_titulo,
        mf.raw -> 'detalhes' ->> 'cGrupo'                        AS grupo,
        mf.raw -> 'detalhes' ->> 'dDtPagamento'                  AS data_pagamento,
        mf.raw -> 'detalhes' ->> 'dDtPrevisao'                   AS data_previsao,
        mf.codigo_categoria                                       AS codigo_categoria,
        mf.raw -> 'detalhes' ->> 'cStatus'                       AS status_mov,
        COALESCE(
            NULLIF(mf.raw -> 'resumo'   ->> 'nValLiquido',  '')::NUMERIC,
            NULLIF(mf.raw -> 'resumo'   ->> 'nValPago',     '')::NUMERIC,
            NULLIF(mf.raw -> 'detalhes' ->> 'nValorMovCC',  '')::NUMERIC
        )                                                         AS valor_titulo_mov_ext,
        EXTRACT(YEAR  FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS ano,
        EXTRACT(MONTH FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))::INT AS mes,
        'mov'::TEXT                                               AS mov_ou_extrato,
        (SELECT dep ->> 'cCodDepartamento'
           FROM jsonb_array_elements(
                    CASE WHEN jsonb_typeof(mf.raw -> 'departamentos') = 'array'
                         THEN mf.raw -> 'departamentos'
                         ELSE '[]'::JSONB END) AS dep
          LIMIT 1)                                                AS departamento_mov,
        NULL::TEXT                                                AS descricao_categoria_ext,
        NULL::TEXT                                                AS cliente_ext
    FROM %s.movimentos_financeiros mf
    WHERE mf.raw -> 'detalhes' ->> 'cGrupo' IN ('CONTA_CORRENTE_REC','CONTA_CORRENTE_PAG')
      AND NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento','') IS NOT NULL
      AND EXTRACT(YEAR FROM TO_DATE(NULLIF(mf.raw -> 'detalhes' ->> 'dDtPagamento',''), 'DD/MM/YYYY'))
          < EXTRACT(YEAR FROM CURRENT_DATE)
    ORDER BY mf.empresa_id, COALESCE(mf.codigo_titulo::TEXT, 'mov:' || mf.id::TEXT), mf.id DESC)
),
cp_categorias AS (
    SELECT
        cp.empresa_id,
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
        cr.empresa_id,
        cr.codigo_lancamento::TEXT,
        cr.valor_documento,
        (cat_elem ->> 'valor')::NUMERIC,
        (cat_elem ->> 'percentual')::NUMERIC,
        cat_elem ->> 'codigo_categoria',
        'contas_a_receber'::TEXT
    FROM %s.contas_receber cr,
         LATERAL jsonb_array_elements(cr.raw -> 'categorias') AS cat_elem
),
cp_distribuicao AS (
    SELECT
        cp.empresa_id,
        cp.codigo_lancamento::TEXT                     AS id,
        (dist_elem ->> 'nValDep')::NUMERIC             AS valor_distribuido,
        (dist_elem ->> 'nPerDep')::NUMERIC             AS percentual_distribuicao,
        dist_elem ->> 'cCodDep'                        AS codigo_departamento
    FROM %s.contas_pagar cp,
         LATERAL jsonb_array_elements(cp.raw -> 'distribuicao') AS dist_elem
    UNION ALL
    SELECT
        cr.empresa_id,
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
        COALESCE(d.codigo_departamento, m.departamento_mov) AS codigo_departamento_join,
        d.percentual_distribuicao                      AS percentual_distribuicao_join,
        d.valor_distribuido                            AS valor_distribuido_join,
        cc.fluxo_caixa                                 AS conta_considerada,
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
        -- ABS só no ext, cujo valor agora carrega sinal. O dashboard soma receita e
        -- despesa como grandezas positivas e separa pelo ajuste_receita_despesa.
        -- No mov o valor já é positivo, então o ramo fica idêntico ao anterior.
        CASE WHEN m.mov_ou_extrato = 'ext' THEN ABS(m.valor_titulo_mov_ext)
             ELSE m.valor_titulo_mov_ext END
            * COALESCE(c.percentual_categoria, 100) / 100.0 AS valor_final,
        CASE
            WHEN m.mov_ou_extrato = 'ext'                                                   THEN 1
            WHEN m.mov_ou_extrato = 'mov'
             AND m.grupo IN ('CONTA_CORRENTE_PAG','CONTA_CORRENTE_REC')                     THEN 1
            ELSE 0
        END                                            AS movimento_considerado,
        -- Fronteira temporal: passado vem do realizado (mov), futuro vem do previsto
        -- (ext). Sem isso, um pagamento agendado com data futura no mov somaria em
        -- duplicidade com a provisão correspondente do extrato.
        CASE
            WHEN m.mov_ou_extrato = 'mov'
             AND TO_DATE(NULLIF(m.data_pagamento,''), 'DD/MM/YYYY') <  CURRENT_DATE THEN 1
            WHEN m.mov_ou_extrato = 'ext'
             AND TO_DATE(NULLIF(m.data_pagamento,''), 'DD/MM/YYYY') >= CURRENT_DATE THEN 1
            ELSE 0
        END                                            AS considerar_mov_ext,
        -- O extrato traz cDesCategoria e cRazCliente prontos; servem de fallback
        -- quando a categoria ou o cliente ainda não foram sincronizados.
        COALESCE(cat_final.descricao, m.descricao_categoria_ext) AS descricao_categoria_final,
        -- Título usa a distribuição de contas_pagar/receber; lançamento avulso usa o
        -- array departamentos do próprio movimento.
        CASE
            WHEN COALESCE(d.codigo_departamento, m.departamento_mov) IS NULL THEN 'Sem departamento'
            ELSE dept.descricao
        END                                            AS departamento_final,
        COALESCE(cli.nome_fantasia, m.cliente_ext, 'Cliente não informado') AS cliente_final,
        emp.nome                                       AS nome_empresa,
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
    LEFT JOIN cp_categorias   c    ON m.codigo_titulo = c.id   AND m.empresa_id = c.empresa_id
    LEFT JOIN cp_distribuicao d    ON m.codigo_titulo = d.id   AND m.empresa_id = d.empresa_id
    LEFT JOIN %s.contas_correntes cc
           ON cc.codigo_conta_corrente::TEXT = m.codigo_conta_corrente
          AND cc.empresa_id = m.empresa_id
    LEFT JOIN categorias_processadas cat_sup
           ON LEFT(COALESCE(c.codigo_categoria, m.codigo_categoria), 4) = cat_sup.codigo
          AND cat_sup.empresa_id = m.empresa_id
    LEFT JOIN categorias_processadas cat_final
           ON COALESCE(c.codigo_categoria, m.codigo_categoria) = cat_final.codigo
          AND cat_final.empresa_id = m.empresa_id
    LEFT JOIN %s.departamentos dept
           ON COALESCE(d.codigo_departamento, m.departamento_mov) = dept.codigo
          AND dept.empresa_id = m.empresa_id
    LEFT JOIN %s.clientes cli
           ON cli.codigo_cliente_omie::TEXT = m.codigo_cliente
          AND cli.empresa_id = m.empresa_id
    LEFT JOIN _etl.empresas emp
           ON emp.id = m.empresa_id
)
SELECT
    empresa_id, nome_empresa, codigo_conta_corrente, codigo_cliente,
    codigo_titulo, grupo, data_pagamento, data_previsao, codigo_categoria,
    status_mov, valor_titulo_mov_ext, ano, mes, mov_ou_extrato,
    categoria_join, percentual_categoria, origem, codigo_departamento_join,
    percentual_distribuicao_join, valor_distribuido_join, conta_considerada,
    cod_categoria_final_superior, cod_categoria_final, percentual_cat_final,
    receita_despesa, ajuste_receita_despesa, valor_final,
    movimento_considerado, considerar_mov_ext,
    descricao_categoria_superior, descricao_categoria_final,
    departamento_final, cliente_final
FROM movimentos_processados
WHERE movimento_considerado = 1 AND considerar_mov_ext = 1
WITH NO DATA`,
			safe,           // matvw_gerencial_historico
			safe,           // categorias_processadas: categorias
			safe,           // movimentos_financeiros (sem ramo ext: extrato só tem futuro)
			safe, safe,     // cp_categorias: contas_pagar, contas_receber
			safe, safe,     // cp_distribuicao: contas_pagar, contas_receber
			safe,           // movimentos_processados: contas_correntes
			safe,           // movimentos_processados: departamentos
			safe,           // movimentos_processados: clientes
		),

		// Índices: historico (dataset grande — ano é filtro primário)
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_hist_ano ON %s.matvw_gerencial_historico (ano)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_hist_ano_mes ON %s.matvw_gerencial_historico (ano, mes)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_hist_receita ON %s.matvw_gerencial_historico (receita_despesa)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_hist_categoria ON %s.matvw_gerencial_historico (cod_categoria_final)", schemaName, safe),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_hist_empresa ON %s.matvw_gerencial_historico (empresa_id)", schemaName, safe),
	}

	for i, stmt := range stmts {
		if _, err := p.pool.Exec(ctx, stmt); err != nil {
			// Inclui um trecho do statement: sem isso, um erro de sintaxe numa das
			// views gerenciais não diz qual das dezenas de statements falhou.
			trecho := stmt
			if len(trecho) > 200 {
				trecho = trecho[:200] + "..."
			}
			return fmt.Errorf("db.Provisioner.ProvisionSchema [%s] statement %d (%s): %w", schemaName, i, trecho, err)
		}
	}

	if err := p.markProvisioned(ctx, schemaName); err != nil {
		// Não fatal — schema está correto, só o registro de versão falhou
		fmt.Printf("db.Provisioner.ProvisionSchema: markProvisioned falhou para %s: %v\n", schemaName, err)
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
	// Remove entrada de versão ao dropar o schema
	_, _ = p.pool.Exec(ctx, `DELETE FROM _etl.schema_versions WHERE schema_name = $1`, schemaName)
	return nil
}
