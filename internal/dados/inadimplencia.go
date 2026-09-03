package dados

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Inadimplência: títulos vencidos e não pagos.
//
// Por que não sai da view materializada — que é de onde vem todo o resto do
// dashboard: a matvw é o fluxo de caixa REALIZADO mais as provisões do extrato.
// Um título vencido e não pago não está em nenhum dos dois canais. Conferido em
// produção: dos 35 títulos ATRASADO, zero aparecem na matvw, então somar não
// duplica nada.
//
// A projeção e as regras de rateio abaixo copiam a matvw de propósito. Se
// divergirem, a mesma categoria some com valores diferentes entre as abas.

// statusAtrasado e o rotulo que identifica a linha vencida. Constante porque o
// valor viaja da projecao SQL ate a classificacao do resumo e a pill do
// frontend: escrito solto em cada lugar, uma divergencia de caixa faria o
// atrasado voltar a se esconder dentro do previsto, sem erro nenhum.
const statusAtrasado = "Atrasado"

// colunasInadimplencia devolve exatamente o mesmo formato de colunasFluxo, para
// que as linhas possam ser anexadas à mesma lista sem tratamento especial.
//
// `tipo` é constante por tabela: contas_receber é sempre receita, contas_pagar
// sempre despesa. Não há a ambiguidade que a matvw resolve por status.
const projecaoInadimplencia = `
	EXTRACT(DAY FROM t.data_vencimento)::INT                       AS dia,
	TO_CHAR(t.data_vencimento, 'DD/MM/YYYY')                       AS data,
	COALESCE(NULLIF(cli.razao_social, ''), 'Não informado')         AS descricao,
	%s                                                             AS tipo,
	COALESCE(NULLIF(cat_fim.descricao, ''), 'Sem categoria')       AS categoria,
	t.valor_documento * COALESCE(rateio.percentual, 100) / 100.0   AS valor,
	'` + statusAtrasado + `'                                        AS status,
	FALSE                                                          AS realizado`

/*
fonteInadimplencia monta o FROM com os mesmos rateios da view materializada.

Três assimetrias vieram de lá e precisam ser mantidas:

 1. Categoria RATEIA o valor (percentual / 100), departamento NÃO. A própria
    matvw registra o porquê: os percentuais de categoria somam 100 e portanto
    dividem o título, enquanto duas linhas de distribuição repetiriam o valor
    cheio. Por isso o departamento é deduplicado para um por título.

 2. O join dos arrays é LEFT JOIN LATERAL ... ON TRUE, e não LATERAL puro:
    título sem o array `categorias` desapareceria, e é justamente um
    inadimplente sumindo da tela. O COALESCE(percentual, 100) devolve o valor
    cheio nesse caso, igual à matvw.

 3. A categoria do FILTRO é a superior — LEFT(codigo, 4) resolvido contra a
    tabela categorias —, enquanto a EXIBIDA é a final. É assim na matvw; usar a
    final no filtro faria uma categoria marcada esconder inadimplência que
    deveria aparecer.

A conta corrente vive só em raw->>'id_conta_corrente': o ETL não a grava em
coluna tipada. Comparada como texto, que é a forma que a matvw expõe.
*/
func fonteInadimplencia(safe, tabela string) string {
	return fmt.Sprintf(`
		%[1]s.%[2]s t

		LEFT JOIN LATERAL jsonb_array_elements(t.raw -> 'categorias') AS cat_elem ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				(cat_elem ->> 'percentual')::NUMERIC AS percentual,
				cat_elem ->> 'codigo_categoria'      AS codigo
		) AS rateio ON TRUE

		LEFT JOIN %[1]s.categorias cat_fim
		       ON cat_fim.empresa_id = t.empresa_id
		      AND cat_fim.codigo     = COALESCE(rateio.codigo, t.codigo_categoria)

		LEFT JOIN %[1]s.categorias cat_sup
		       ON cat_sup.empresa_id = t.empresa_id
		      AND cat_sup.codigo     = LEFT(COALESCE(rateio.codigo, t.codigo_categoria), 4)

		LEFT JOIN %[1]s.clientes cli
		       ON cli.empresa_id          = t.empresa_id
		      AND cli.codigo_cliente_omie = t.codigo_cliente

		LEFT JOIN LATERAL (
			SELECT DISTINCT ON (dist_elem ->> 'cCodDep') dist_elem ->> 'cCodDep' AS codigo
			FROM jsonb_array_elements(t.raw -> 'distribuicao') AS dist_elem
			ORDER BY dist_elem ->> 'cCodDep'
			LIMIT 1
		) AS depto ON TRUE
	`, safe, tabela)
}

/*
buildFiltroInadimplencia traduz os filtros globais para as colunas dos títulos.

Não reaproveita buildFiltro de propósito: aquele escreve contra as colunas da
view materializada (ano, mes, cliente_final, descricao_categoria_superior…), que
não existem aqui. O que se mantém é o SIGNIFICADO de cada filtro, não o SQL.

Ano e mês incidem sobre data_vencimento, e não sobre a data de pagamento: um
título atrasado nunca foi pago, e o que interessa é onde ele deveria ter sido.

A exclusão de categorias usa a superior, igual ao resto do dashboard, senão a
"Transferência" oculta por padrão voltaria a aparecer só na inadimplência.
*/
func buildFiltroInadimplencia(p DashboardParams, mes int) (string, []any) {
	conds := []string{"UPPER(t.status_titulo) = 'ATRASADO'", "t.data_vencimento IS NOT NULL"}
	args := []any{}
	idx := 1

	add := func(cond string, val any) {
		conds = append(conds, fmt.Sprintf(cond, idx))
		args = append(args, val)
		idx++
	}

	add("EXTRACT(YEAR FROM t.data_vencimento) = $%d", p.Ano)
	if mes >= 1 && mes <= 12 {
		add("EXTRACT(MONTH FROM t.data_vencimento) = $%d", mes)
	}
	if len(p.Empresas) > 0 {
		add("t.empresa_id::text = ANY($%d)", p.Empresas)
	}
	if len(p.ContasCorrentes) > 0 {
		add("NULLIF(t.raw ->> 'id_conta_corrente','') = ANY($%d)", p.ContasCorrentes)
	}
	if len(p.Departamentos) > 0 {
		add("depto.codigo = ANY($%d)", p.Departamentos)
	}
	if len(p.Categorias) > 0 {
		add("cat_sup.descricao = ANY($%d)", p.Categorias)
	}
	if len(p.CategoriasExcluir) > 0 {
		add("NOT (COALESCE(cat_sup.descricao, '') = ANY($%d))", p.CategoriasExcluir)
	}
	if p.Cliente != "" {
		add("cli.razao_social ILIKE $%d", "%"+p.Cliente+"%")
	}

	return strings.Join(conds, "\n\t\t  AND "), args
}

// queryInadimplencia devolve os títulos vencidos das duas tabelas, já rateados,
// no formato de FluxoTransacao.
//
// Erro aqui NÃO derruba a aba: quem chama degrada para a lista sem inadimplência.
// A tabela pode não existir num grupo cujo schema é de versão anterior, e perder
// o fluxo inteiro por causa disso seria pior que perder o acréscimo.
func queryInadimplencia(ctx context.Context, pool *pgxpool.Pool, schema string, p DashboardParams, mes int) ([]FluxoTransacao, error) {
	safe := pgx.Identifier{schema}.Sanitize()

	var todas []FluxoTransacao
	for _, f := range []struct {
		tabela string
		tipo   string
	}{
		{"contas_receber", "'receita'"},
		{"contas_pagar", "'despesa'"},
	} {
		where, args := buildFiltroInadimplencia(p, mes)
		sql := fmt.Sprintf(`
			SELECT %s
			FROM %s
			WHERE %s
			ORDER BY t.data_vencimento, t.valor_documento DESC
		`, fmt.Sprintf(projecaoInadimplencia, f.tipo), fonteInadimplencia(safe, f.tabela), where)

		linhas, err := scanTransacoes(ctx, pool, sql, args)
		if err != nil {
			return nil, fmt.Errorf("dados.queryInadimplencia %s: %w", f.tabela, err)
		}
		todas = append(todas, linhas...)
	}
	return todas, nil
}
