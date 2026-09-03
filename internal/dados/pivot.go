package dados

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PivotLinha é uma folha da hierarquia tipo → categoria superior → categoria final
// → cliente, já com os 12 meses do ano.
//
// O pivot acontece no SQL, não no frontend: agrupar por mês multiplicaria as linhas
// por 12 e obrigaria o cliente a cruzar tudo. Assim cada combinação vem uma vez só,
// e a árvore é montada somando os filhos, sem consulta adicional por expansão.
type PivotLinha struct {
	Tipo              string      `json:"tipo"` // "receita" | "despesa"
	CategoriaSuperior string      `json:"categoria_superior"`
	CategoriaFinal    string      `json:"categoria_final"`
	Cliente           string      `json:"cliente"`
	Meses             [12]float64 `json:"meses"`
	Total             float64     `json:"total"`
}

type PivotResponse struct {
	Ano    int          `json:"ano"`
	Linhas []PivotLinha `json:"linhas"`

	// ResultadoMes é receita MENOS despesa de cada mês, não a soma das linhas.
	//
	// Antes o rodapé somava tudo no mesmo acumulador. Como receita e despesa são
	// ambas grandezas positivas, o resultado era receita + despesa — um número sem
	// significado financeiro.
	//
	// Não inclui o saldo das contas correntes: o card RESULTADO da Visão Geral é
	// receita − despesa + saldo, então os dois diferem pelo caixa de abertura, de
	// propósito. Um é resultado do período; o outro, posição de caixa.
	ResultadoMes   [12]float64 `json:"resultado_mes"`
	ResultadoTotal float64     `json:"resultado_total"`
	// MesCorte separa realizado de previsto: a matvw usa considerar_mov_ext, então
	// meses anteriores vêm de movimentos realizados e os seguintes, de provisões do
	// extrato. Sem marcar a fronteira, a tela compara naturezas diferentes.
	MesCorte int `json:"mes_corte"`
}

const totalMeses = 12

// QueryPivot devolve a matriz de resultado por categoria e cliente ao longo do ano.
func QueryPivot(ctx context.Context, pool *pgxpool.Pool, p DashboardParams) (*PivotResponse, error) {
	schema, err := schemaForGrupo(ctx, pool, p.GrupoID)
	if err != nil {
		return nil, err
	}

	safe := pgx.Identifier{schema}.Sanitize()
	view := viewName(p.Ano)
	where, args := buildFiltroAno(p)

	// FILTER (WHERE mes = N) é o pivot: uma coluna por mês, uma linha por combinação.
	sql := fmt.Sprintf(`
		SELECT
			-- Deriva de ajuste_receita_despesa, e não da coluna textual
			-- receita_despesa, que é NULL quando a classificação não resolve.
			-- O dashboard usa o ajuste, cujo ELSE manda o não classificado para
			-- despesa; agrupar pelo texto criaria um terceiro grupo aqui e faria
			-- as duas abas divergirem no total.
			CASE ajuste_receita_despesa WHEN 1 THEN 'receita' ELSE 'despesa' END AS tipo,
			COALESCE(NULLIF(descricao_categoria_superior, ''), 'Sem categoria') AS cat_superior,
			COALESCE(NULLIF(descricao_categoria_final, ''),    'Sem categoria') AS cat_final,
			COALESCE(NULLIF(cliente_final, ''), 'Não informado')   AS cliente,
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  1), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  2), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  3), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  4), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  5), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  6), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  7), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  8), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes =  9), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes = 10), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes = 11), 0),
			COALESCE(SUM(valor_final) FILTER (WHERE mes = 12), 0),
			COALESCE(SUM(valor_final), 0)                          AS total
		FROM %s.%s
		WHERE %s
		GROUP BY 1, 2, 3, 4
		ORDER BY 1, 2, 3, 4
	`, safe, view, where)

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryPivot: %w", err)
	}
	defer rows.Close()

	resp := &PivotResponse{Ano: p.Ano, Linhas: []PivotLinha{}}

	for rows.Next() {
		var l PivotLinha
		dest := []any{&l.Tipo, &l.CategoriaSuperior, &l.CategoriaFinal, &l.Cliente}
		for i := range l.Meses {
			dest = append(dest, &l.Meses[i])
		}
		dest = append(dest, &l.Total)

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("dados.QueryPivot scan: %w", err)
		}

		// Resultado no servidor: se a tela calculasse por conta própria,
		// arredondamento divergiria do que o banco reporta nos cards.
		//
		// As linhas seguem com a magnitude positiva — quem carrega o sinal é o
		// rodapé, onde despesa subtrai.
		sinal := 1.0
		if l.Tipo == "despesa" {
			sinal = -1.0
		}
		for i, v := range l.Meses {
			resp.ResultadoMes[i] += sinal * v
		}
		resp.ResultadoTotal += sinal * l.Total
		resp.Linhas = append(resp.Linhas, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dados.QueryPivot rows: %w", err)
	}

	resp.MesCorte = mesCorte(p.Ano)
	return resp, nil
}

// mesCorte devolve o primeiro mês projetado do ano consultado.
// Ano passado: tudo realizado (13). Ano futuro: tudo previsto (1).
func mesCorte(ano int) int {
	agora := time.Now()
	switch {
	case ano < agora.Year():
		return totalMeses + 1
	case ano > agora.Year():
		return 1
	default:
		return int(agora.Month())
	}
}
