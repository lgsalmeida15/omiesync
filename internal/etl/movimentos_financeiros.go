package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/worker"
)

const mfMaxRetries = 3

// isMFRetryable detecta erros transitórios do servidor Omie que valem retry.
func isMFRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "soap-env:server") ||
		strings.Contains(msg, "broken response") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline") ||
		strings.Contains(msg, "connection reset")
}

// mfPaginacaoParams usa campos específicos do endpoint /financas/mf/ (nPagina/nRegPorPagina
// em vez do padrão pagina/registros_por_pagina), e filtros de data por inclusão e alteração.
type mfPaginacaoParams struct {
	NPagina             int    `json:"nPagina"`
	NRegPorPagina       int    `json:"nRegPorPagina"`
	TpLancamento        string `json:"cTpLancamento,omitempty"`
	DtIncDe             string `json:"dDtIncDe,omitempty"`
	DtIncAte            string `json:"dDtIncAte,omitempty"`
	DtAltDe             string `json:"dDtAltDe,omitempty"`
	DtAltAte            string `json:"dDtAltAte,omitempty"`
	ExibirDepartamentos string `json:"cExibirDepartamentos"`
}

type OmieMovimentoDepartamento struct {
	CodigoDepartamento string  `json:"cCodDepartamento"`
	Percentual         float64 `json:"nDistrPercentual"`
	Valor              float64 `json:"nDistrValor"`
}

// OmieMovimentoDetalhes — campos do bloco "detalhes" na resposta /financas/mf/.
//
// Atenção: nCodTitulo NÃO identifica o movimento. Uma baixa em lote liquida N títulos
// com um único nCodMovCC, e lançamentos avulsos (tarifa, transferência) chegam com
// nCodTitulo = 0. Nenhum dos dois serve como chave — por isso a tabela é recarregada
// por completo a cada sync.
type OmieMovimentoDetalhes struct {
	CodigoTitulo        int64   `json:"nCodTitulo"`
	CodigoTitRepet      int64   `json:"nCodTitRepet"`
	CodigoMovCC         int64   `json:"nCodMovCC"`
	CodigoMovCCRepet    int64   `json:"nCodMovCCRepet"`
	DataRegistro        string  `json:"dDtRegistro"`
	DataEmissao         string  `json:"dDtEmissao"`
	DataVencimento      string  `json:"dDtVenc"`
	DataPagamento       string  `json:"dDtPagamento"`
	ValorTitulo         float64 `json:"nValorTitulo"`
	ValorMovCC          float64 `json:"nValorMovCC"`
	Status              string  `json:"cStatus"`
	Grupo               string  `json:"cGrupo"`
	CodigoContaCorrente int64   `json:"nCodCC"`
	CodigoCategoria     string  `json:"cCodCateg"`
	NumeroTitulo        string  `json:"cNumTitulo"`
	Natureza            string  `json:"cNatureza"`
	CodigoCliente       int64   `json:"nCodCliente"`
}

type OmieMovimentoResumo struct {
	ValorPago   float64 `json:"nValPago"`
	ValorAberto float64 `json:"nValAberto"`
	Liquidado   string  `json:"cLiquidado"`
}

// OmieMovimento — departamentos está no nível raiz do objeto, não dentro de detalhes.
type OmieMovimento struct {
	Departamentos []OmieMovimentoDepartamento `json:"departamentos"`
	Detalhes      OmieMovimentoDetalhes       `json:"detalhes"`
	Resumo        OmieMovimentoResumo         `json:"resumo"`
}

type listarMovimentosResp struct {
	NPagina       int             `json:"nPagina"`
	NTotPaginas   int             `json:"nTotPaginas"`
	NTotRegistros int             `json:"nTotRegistros"`
	Movimentos    []OmieMovimento `json:"movimentos"`
}

type MovimentosFinanceirosExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewMovimentosFinanceirosExecutor(pool *pgxpool.Pool, log zerolog.Logger) *MovimentosFinanceirosExecutor {
	return &MovimentosFinanceirosExecutor{pool: pool, log: log.With().Str("executor", "movimentos_financeiros").Logger()}
}

func (e *MovimentosFinanceirosExecutor) Nome() string { return "movimentos_financeiros" }

// buildMFParams monta os parâmetros do endpoint /financas/mf/.
//
// Incremental e full são idênticos: sem filtro de data, a API devolve todos os
// movimentos de conta corrente pela paginação natural. Os campos dDtIncDe/dDtIncAte/
// dDtAltDe/dDtAltAte têm omitempty e simplesmente não são serializados.
//
// A janela incremental de D-2 foi removida: como nCodTitulo não identifica o movimento,
// não havia como mesclar um recorte parcial sem perder registros. Cada sync recarrega
// tudo. A estratégia de incremental será redefinida a partir da análise do dado completo.
func buildMFParams(pagina, pageSize int, _ worker.SyncOptions) mfPaginacaoParams {
	return mfPaginacaoParams{
		NPagina:             pagina,
		NRegPorPagina:       pageSize,
		ExibirDepartamentos: "S",
		TpLancamento:        "CC",
	}
}

func (e *MovimentosFinanceirosExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	_ = rep.Start(ctx, jobID, "movimentos_financeiros")
	pagina := 1
	total := 0
	totalEsperado := 0

	var acumulados []OmieMovimento
	var acumuladosRaw []json.RawMessage

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 500
	}

	for {
		params := buildMFParams(pagina, pageSize, opts)

		var respRaw map[string]json.RawMessage
		var fetchErr error
		for attempt := 0; attempt < mfMaxRetries; attempt++ {
			if attempt > 0 {
				wait := time.Duration(attempt*10) * time.Second
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
			}
			fetchErr = client.CallPublic(ctx, cfg.EndpointPath, cfg.Action, params, &respRaw)
			if fetchErr == nil || !isMFRetryable(fetchErr) {
				break
			}
		}
		if fetchErr != nil {
			// Omie retorna SOAP-ENV:Client-500 "Não existem registros para a página" quando
			// o filtro de data não encontra nenhum dado — tratar como sucesso vazio.
			if omie.IsSemRegistros(fetchErr) {
				e.log.Info().Str("job_id", jobID).Bool("full", opts.Full).
					Msg("movimentos_financeiros: nenhum registro no período (sem registros)")
				break
			}
			_ = rep.Fail(ctx, jobID, "movimentos_financeiros", fetchErr, client.LastMaskedPayload, string(client.LastResponseMeta))
			return fmt.Errorf("etl.movimentos_financeiros fetch página %d: %w", pagina, fetchErr)
		}

		// Extrai paginação
		var nTotPaginas, nTotRegistros int
		if v, ok := respRaw["nTotPaginas"]; ok {
			_ = json.Unmarshal(v, &nTotPaginas)
		}
		if v, ok := respRaw["nTotRegistros"]; ok {
			_ = json.Unmarshal(v, &nTotRegistros)
		}

		// Extrai movimentos tipados + JSON bruto original (para coluna raw)
		var movimentos []OmieMovimento
		var movimentosRaw []json.RawMessage
		if v, ok := respRaw["movimentos"]; ok {
			_ = json.Unmarshal(v, &movimentos)
			_ = json.Unmarshal(v, &movimentosRaw)
		}

		e.log.Debug().
			Str("job_id", jobID).
			Int("pagina", pagina).
			Int("nTotPaginas", nTotPaginas).
			Int("nTotRegistros", nTotRegistros).
			Int("movimentos_na_pagina", len(movimentos)).
			Msg("movimentos_financeiros: resposta recebida")

		// Omie retorna nTotPaginas=0 quando não há registros no período.
		if nTotPaginas == 0 && pagina == 1 {
			e.log.Info().Str("job_id", jobID).Bool("full", opts.Full).
				Msg("movimentos_financeiros: nenhum registro no período (nTotPaginas=0)")
			break
		}

		// Acumula em memória: a gravação só acontece depois que todas as páginas
		// chegaram, para que o DELETE + INSERT caibam numa única transação curta.
		acumulados = append(acumulados, movimentos...)
		acumuladosRaw = append(acumuladosRaw, movimentosRaw...)
		if nTotRegistros > 0 {
			totalEsperado = nTotRegistros
		}

		total += len(movimentos)
		_ = rep.UpdatePage(ctx, jobID, "movimentos_financeiros", pagina, nTotPaginas, total, nTotRegistros, client.LastMaskedPayload, client.LastResponseMeta)
		_ = rep.Heartbeat(ctx)

		if pagina >= nTotPaginas {
			break
		}
		pagina++
	}

	// Troca atômica: os leitores enxergam o dado anterior até o COMMIT, e uma falha
	// em qualquer ponto faz rollback sem destruir o que já existia.
	gravados, err := replaceMovimentos(ctx, e.pool, schema, opts.EmpresaID, acumulados, acumuladosRaw)
	if err != nil {
		_ = rep.Fail(ctx, jobID, "movimentos_financeiros", err, client.LastMaskedPayload, string(client.LastResponseMeta))
		return fmt.Errorf("etl.movimentos_financeiros gravação: %w", err)
	}

	// Reconciliação: sem chave de negócio, divergência silenciosa é o risco principal.
	if totalEsperado > 0 && gravados != totalEsperado {
		e.log.Warn().
			Str("job_id", jobID).
			Int("nTotRegistros_omie", totalEsperado).
			Int("gravados", gravados).
			Int("diferenca", totalEsperado-gravados).
			Msg("movimentos_financeiros: divergência entre o total informado pelo Omie e o gravado")
	}

	_ = rep.Done(ctx, jobID, "movimentos_financeiros", gravados)
	return nil
}

// ExecutePage existe apenas para satisfazer worker.Executor. Este executor NÃO suporta
// sub-jobs por página: a gravação é uma troca completa por empresa (DELETE + COPY numa
// transação), então cada página escrita isoladamente apagaria as anteriores.
//
// Falha explícita em vez de silenciosa — se um dia o PageWorker for ligado, o erro
// aparece no job em vez de a tabela ficar só com a última página.
func (e *MovimentosFinanceirosExecutor) ExecutePage(_ context.Context, _ *omie.Client, _ string, _ worker.SyncOptions, _ int, _ *omie_config.EndpointConfig) (int, error) {
	return 0, fmt.Errorf("etl.movimentos_financeiros: executor não suporta sub-jobs por página; usar Execute (carga completa transacional)")
}

// replaceMovimentos troca todo o conjunto de movimentos da empresa por `items`, dentro
// de uma única transação. Retorna quantas linhas foram gravadas.
//
// A tabela não tem chave de negócio (ver comentário em OmieMovimentoDetalhes), então a
// corretude depende inteiramente desta transação: sem ela, um sync repetido duplicaria
// tudo. DELETE e COPY precisam permanecer no mesmo BEGIN.
//
// O escopo é SEMPRE por empresa_id — o schema é do grupo e abriga várias empresas.
// Um TRUNCATE aqui apagaria os dados das demais.
func replaceMovimentos(
	ctx context.Context,
	pool *pgxpool.Pool,
	schema string,
	empresaID string,
	items []OmieMovimento,
	raws []json.RawMessage,
) (int, error) {
	// Guarda: resposta vazia não apaga o que já existe. Uma falha transitória do Omie
	// que devolva zero registros zeraria a empresa inteira.
	if len(items) == 0 {
		return 0, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("replaceMovimentos begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op após o commit

	if _, err := tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s.movimentos_financeiros WHERE empresa_id = $1", schema),
		empresaID,
	); err != nil {
		return 0, fmt.Errorf("replaceMovimentos delete: %w", err)
	}

	rows := make([][]any, 0, len(items))
	for i, it := range items {
		d := it.Detalhes

		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}

		// nCodTitulo = 0 significa lançamento avulso (tarifa, transferência): sem título.
		// Antes esses registros eram descartados em silêncio.
		var codigoTitulo *int64
		if d.CodigoTitulo != 0 {
			v := d.CodigoTitulo
			codigoTitulo = &v
		}

		// Lançamento avulso não traz nValorTitulo — o valor vem em nValorMovCC.
		valor := d.ValorTitulo
		if valor == 0 {
			valor = d.ValorMovCC
		}

		rows = append(rows, []any{
			empresaID,
			codigoTitulo,
			d.NumeroTitulo,   // cNumTitulo
			d.CodigoMovCC,    // nCodMovCC
			d.CodigoMovCCRepet,
			d.CodigoTitRepet,
			// Ponteiro, não string: o COPY binário não aceita "" numa coluna DATE.
			parseOmieDatePtr(d.DataRegistro), // dDtRegistro — data de criação do lançamento
			valor,
			d.Grupo + " " + d.Natureza, // ex: "CONTA_A_RECEBER R"
			d.CodigoContaCorrente,
			d.CodigoCategoria,
			raw,
		})
	}

	copied, err := tx.CopyFrom(ctx,
		pgx.Identifier{schema, "movimentos_financeiros"},
		[]string{
			"empresa_id", "codigo_titulo", "numero_titulo", "codigo_mov_cc",
			"codigo_mov_cc_repet", "codigo_titrepet", "data_lancamento", "valor",
			"tipo", "codigo_conta_corrente", "codigo_categoria", "raw",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("replaceMovimentos copy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("replaceMovimentos commit: %w", err)
	}

	return int(copied), nil
}
