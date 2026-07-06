package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
type OmieMovimentoDetalhes struct {
	CodigoLancamento    int64   `json:"nCodTitulo"`
	CodigoTitRepet      int64   `json:"nCodTitRepet"`
	DataRegistro        string  `json:"dDtRegistro"`
	DataEmissao         string  `json:"dDtEmissao"`
	DataVencimento      string  `json:"dDtVenc"`
	DataPagamento       string  `json:"dDtPagamento"`
	ValorTitulo         float64 `json:"nValorTitulo"`
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

// buildMFParams monta os parâmetros de data para o endpoint /financas/mf/.
// Full → período amplo (01/01/2000 até hoje) para capturar todos os movimentos.
// Incremental → apenas registros incluídos ou alterados hoje.
func buildMFParams(pagina, pageSize int, opts worker.SyncOptions) mfPaginacaoParams {
	p := mfPaginacaoParams{
		NPagina:             pagina,
		NRegPorPagina:       pageSize,
		ExibirDepartamentos: "S",
		TpLancamento:        "CC",
	}
	if !opts.Full {
		// Incremental: janela de D-2 até hoje — captura lançamentos com até 2 dias de atraso
		hoje := time.Now().Format("02/01/2006")
		doisDiasAtras := time.Now().AddDate(0, 0, -2).Format("02/01/2006")
		p.DtIncDe = doisDiasAtras
		p.DtIncAte = hoje
		p.DtAltDe = doisDiasAtras
		p.DtAltAte = hoje
	}
	// Full: sem filtro de data — retorna todos os CC via paginação natural da API
	return p
}

func (e *MovimentosFinanceirosExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	_ = rep.Start(ctx, jobID, "movimentos_financeiros")
	pagina := 1
	total := 0

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 50
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

		if len(movimentos) > 0 {
			if err := upsertMovimentos(ctx, e.pool, schema, opts.EmpresaID, movimentos, movimentosRaw); err != nil {
				_ = rep.Fail(ctx, jobID, "movimentos_financeiros", err, client.LastMaskedPayload, string(client.LastResponseMeta))
				return fmt.Errorf("etl.movimentos_financeiros upsert página %d: %w", pagina, err)
			}
		}

		total += len(movimentos)
		_ = rep.UpdatePage(ctx, jobID, "movimentos_financeiros", pagina, nTotPaginas, total, nTotRegistros, client.LastMaskedPayload, client.LastResponseMeta)
		_ = rep.Heartbeat(ctx)

		if pagina >= nTotPaginas {
			break
		}
		pagina++
	}

	_ = rep.Done(ctx, jobID, "movimentos_financeiros", total)
	return nil
}

func (e *MovimentosFinanceirosExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	params := buildMFParams(pagina, pageSize, opts)

	var resp map[string]json.RawMessage
	var fetchErr error
	for attempt := 0; attempt < mfMaxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt*10) * time.Second
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(wait):
			}
		}
		fetchErr = client.CallPublic(ctx, cfg.EndpointPath, cfg.Action, params, &resp)
		if fetchErr == nil || !isMFRetryable(fetchErr) {
			break
		}
	}
	if fetchErr != nil {
		return 0, fetchErr
	}

	dataRaw, ok := resp[cfg.ArrayField]
	if !ok {
		return 0, fmt.Errorf("campo %s não encontrado", cfg.ArrayField)
	}

	var movimentos []OmieMovimento
	if err := json.Unmarshal(dataRaw, &movimentos); err != nil {
		return 0, fmt.Errorf("erro ao decodificar movimentos: %w", err)
	}

	var movimentosRaw []json.RawMessage
	_ = json.Unmarshal(dataRaw, &movimentosRaw)

	if len(movimentos) > 0 {
		if err := upsertMovimentos(ctx, e.pool, schema, opts.EmpresaID, movimentos, movimentosRaw); err != nil {
			return 0, err
		}
	}

	return len(movimentos), nil
}

func upsertMovimentos(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieMovimento, raws []json.RawMessage) error {
	for i, it := range items {
		d := it.Detalhes
		r := it.Resumo
		if d.CodigoLancamento == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.movimentos_financeiros
				(empresa_id, codigo_lancamento, data_lancamento, valor, tipo,
				 codigo_conta_corrente, codigo_categoria, historico, raw, synced_at)
			VALUES ($1, $2, NULLIF($3,'')::DATE, $4, $5, $6, $7, $8, $9, NOW())
			ON CONFLICT (empresa_id, codigo_lancamento) DO UPDATE SET
				data_lancamento       = EXCLUDED.data_lancamento,
				valor                 = EXCLUDED.valor,
				tipo                  = EXCLUDED.tipo,
				codigo_conta_corrente = EXCLUDED.codigo_conta_corrente,
				codigo_categoria      = EXCLUDED.codigo_categoria,
				historico             = EXCLUDED.historico,
				raw                   = EXCLUDED.raw,
				synced_at             = NOW()
		`, schema),
			empresaID,
			d.CodigoLancamento,
			d.DataRegistro,         // dDtRegistro — data de criação do lançamento
			d.ValorTitulo,          // nValorTitulo — valor nominal (sempre preenchido)
			d.Grupo+" "+d.Natureza, // ex: "CONTA_A_RECEBER R"
			d.CodigoContaCorrente,
			d.CodigoCategoria,
			d.NumeroTitulo,
			raw,
		)
		if err != nil {
			return fmt.Errorf("upsertMovimentos [%d]: %w", d.CodigoLancamento, err)
		}
		_ = r // resumo fica no raw; ValorPago não é usado como coluna separada
	}
	return nil
}
