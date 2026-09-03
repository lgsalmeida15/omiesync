package dados

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// FluxoTransacao é um lançamento individual do fluxo de caixa.
//
// Descricao vem de cliente_final: a matvw não guarda texto livre, e o nome do
// cliente ou fornecedor é o identificador mais útil que existe por linha.
type FluxoTransacao struct {
	Dia       int     `json:"dia"`
	Data      string  `json:"data"` // DD/MM/YYYY
	Descricao string  `json:"descricao"`
	Tipo      string  `json:"tipo"` // "receita" | "despesa"
	Categoria string  `json:"categoria"`
	Valor     float64 `json:"valor"`
	Status    string  `json:"status"` // "Recebido" | "Pago" | "Pendente"
	// Realizado separa o que já aconteceu (movimentos) do que é provisão do
	// extrato. Alimenta o filtro de situação da listagem e o marcador do calendário.
	Realizado bool `json:"realizado"`
}

// FluxoResumo separa realizado de previsto em vez de somar tudo em
// "total a receber" / "total a pagar", que seria ambíguo entre o que já entrou e
// o que ainda vai entrar.
type FluxoResumo struct {
	Recebido float64 `json:"recebido"`
	AReceber float64 `json:"a_receber"`
	Pago     float64 `json:"pago"`
	APagar   float64 `json:"a_pagar"`

	// Vencido e não pago, separado do previsto: um título atrasado tem
	// Realizado=false e caía dentro de AReceber/APagar, somado mas invisível.
	// Fora dali, "a receber" volta a significar só o que ainda vai vencer.
	//
	// Zero quando a inadimplência está desligada — a tela usa isso para não
	// exibir a linha.
	AtrasadoReceber float64 `json:"atrasado_receber"`
	AtrasadoPagar   float64 `json:"atrasado_pagar"`

	// Continua somando os seis: separar o atrasado redistribui as linhas acima,
	// mas não muda o total.
	Resultado float64 `json:"resultado"`
}

type FluxoCaixaResponse struct {
	Ano        int              `json:"ano"`
	Mes        int              `json:"mes"`
	Transacoes []FluxoTransacao `json:"transacoes"`
	Resumo     FluxoResumo      `json:"resumo"`
	// ProximosVencimentos ignora o mês selecionado: são as provisões a partir de
	// hoje, para que a tela continue avisando do que vem mesmo ao navegar por
	// meses passados.
	ProximosVencimentos []FluxoTransacao `json:"proximos_vencimentos"`
}

const maxProximosVencimentos = 12

// colunasFluxo é a projeção compartilhada pelas duas consultas.
//
// O tipo sai de ajuste_receita_despesa, e não da coluna textual receita_despesa,
// que é NULL quando a classificação não resolve. É a mesma regra do dashboard e do
// pivot — usar o texto criaria um terceiro grupo e faria as abas divergirem.
const colunasFluxo = `
	EXTRACT(DAY FROM TO_DATE(NULLIF(data_pagamento,''), 'DD/MM/YYYY'))::INT AS dia,
	COALESCE(data_pagamento, '')                                           AS data,
	COALESCE(NULLIF(cliente_final, ''), 'Não informado')                   AS descricao,
	CASE ajuste_receita_despesa WHEN 1 THEN 'receita' ELSE 'despesa' END   AS tipo,
	COALESCE(NULLIF(descricao_categoria_final, ''), 'Sem categoria')       AS categoria,
	valor_final                                                            AS valor,
	CASE
		WHEN mov_ou_extrato = 'ext'        THEN 'Pendente'
		WHEN ajuste_receita_despesa = 1    THEN 'Recebido'
		ELSE                                    'Pago'
	END                                                                    AS status,
	(mov_ou_extrato = 'mov')                                               AS realizado`

// QueryFluxoCaixa devolve os lançamentos do mês, o resumo já totalizado e os
// próximos vencimentos.
func QueryFluxoCaixa(ctx context.Context, pool *pgxpool.Pool, p DashboardParams) (*FluxoCaixaResponse, error) {
	schema, err := schemaForGrupo(ctx, pool, p.GrupoID)
	if err != nil {
		return nil, err
	}
	safe := pgx.Identifier{schema}.Sanitize()

	mes := p.Mes
	if mes < 1 || mes > 12 {
		mes = int(time.Now().Month())
	}

	resp := &FluxoCaixaResponse{
		Ano:                 p.Ano,
		Mes:                 mes,
		Transacoes:          []FluxoTransacao{},
		ProximosVencimentos: []FluxoTransacao{},
	}

	// ── Transações do mês ────────────────────────────────────────────────────
	where, args := buildFiltro(p, nivelTodos)
	args = append(args, mes)
	sql := fmt.Sprintf(`
		SELECT %s
		FROM %s.%s
		WHERE %s
		  AND mes = $%d
		  AND NULLIF(data_pagamento,'') IS NOT NULL
		ORDER BY TO_DATE(data_pagamento, 'DD/MM/YYYY'), valor_final DESC
	`, colunasFluxo, safe, viewName(p.Ano), where, len(args))

	resp.Transacoes, err = scanTransacoes(ctx, pool, sql, args)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryFluxoCaixa transacoes: %w", err)
	}

	// Inadimplência entra ANTES do resumo somar, e por isso não precisa de
	// tratamento em nenhum outro lugar: o resumo é somado desta lista, e o
	// donut, o top 10 e o calendário são derivados dela no navegador. Uma única
	// inserção alimenta todos.
	//
	// Falha aqui degrada em vez de derrubar: sem o acréscimo a aba continua
	// mostrando o fluxo, que é o dado principal.
	if p.Inadimplencia {
		atrasados, errIna := queryInadimplencia(ctx, pool, schema, p, mes)
		if errIna != nil {
			log.Warn().Err(errIna).
				Str("grupo_id", p.GrupoID).
				Msg("dados: inadimplência indisponível; seguindo sem ela")
		} else {
			resp.Transacoes = append(resp.Transacoes, atrasados...)
		}
	}

	for _, t := range resp.Transacoes {
		switch {
		// Atrasado ANTES de Realizado: ele também tem Realizado=false e, sem este
		// ramo primeiro, voltaria a se esconder dentro de AReceber/APagar.
		case t.Status == statusAtrasado && t.Tipo == "receita":
			resp.Resumo.AtrasadoReceber += t.Valor
		case t.Status == statusAtrasado:
			resp.Resumo.AtrasadoPagar += t.Valor
		case t.Tipo == "receita" && t.Realizado:
			resp.Resumo.Recebido += t.Valor
		case t.Tipo == "receita":
			resp.Resumo.AReceber += t.Valor
		case t.Realizado:
			resp.Resumo.Pago += t.Valor
		default:
			resp.Resumo.APagar += t.Valor
		}
	}
	resp.Resumo.Resultado =
		(resp.Resumo.Recebido + resp.Resumo.AReceber + resp.Resumo.AtrasadoReceber) -
			(resp.Resumo.Pago + resp.Resumo.APagar + resp.Resumo.AtrasadoPagar)

	// ── Próximos vencimentos ─────────────────────────────────────────────────
	// Sempre do ano corrente: o extrato só é sincronizado de hoje em diante, e o
	// ramo ext existe apenas na view do ano corrente.
	pProx := p
	pProx.Ano = time.Now().Year()
	whereProx, argsProx := buildFiltro(pProx, nivelTodos)
	sqlProx := fmt.Sprintf(`
		SELECT %s
		FROM %s.matvw_gerencial_ano_corrente
		WHERE %s
		  AND mov_ou_extrato = 'ext'
		  AND NULLIF(data_pagamento,'') IS NOT NULL
		  AND TO_DATE(data_pagamento, 'DD/MM/YYYY') >= CURRENT_DATE
		ORDER BY TO_DATE(data_pagamento, 'DD/MM/YYYY'), valor_final DESC
		LIMIT %d
	`, colunasFluxo, safe, whereProx, maxProximosVencimentos)

	prox, err := scanTransacoes(ctx, pool, sqlProx, argsProx)
	if err != nil {
		// A view do ano corrente pode não estar populada logo após um
		// re-provisionamento. O mês selecionado já carregou, então degrada em vez
		// de derrubar a tela inteira.
		if !isViewNaoPopulada(err) {
			return nil, fmt.Errorf("dados.QueryFluxoCaixa proximos: %w", err)
		}
	} else {
		resp.ProximosVencimentos = prox
	}

	return resp, nil
}

func scanTransacoes(ctx context.Context, pool *pgxpool.Pool, sql string, args []any) ([]FluxoTransacao, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FluxoTransacao{}
	for rows.Next() {
		var t FluxoTransacao
		if err := rows.Scan(&t.Dia, &t.Data, &t.Descricao, &t.Tipo, &t.Categoria, &t.Valor, &t.Status, &t.Realizado); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
