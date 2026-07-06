package etl

import (
	"context"
	"encoding/json"
	"errors"
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

// OmieExtratoMovimento representa um lançamento de extrato (provisão futura).
// O Omie não retorna ID único — usamos DELETE+INSERT por conta/período.
type OmieExtratoMovimento struct {
	Descricao           string  `json:"cDesCliente"`
	DataLancamento      string  `json:"dDataLancamento"`
	ValorDocumento      float64 `json:"nValorDocumento"`
	Saldo               float64 `json:"nSaldo"`
	CodigoContaCorrente int64   // preenchido pelo executor
}

type listarExtratoParams struct {
	CodigoContaCorrente int64  `json:"nCodCC"`
	DataInicial         string `json:"dPeriodoInicial"`
	DataFinal           string `json:"dPeriodoFinal"`
}

type listarExtratoResp struct {
	ListaMovimentos []OmieExtratoMovimento `json:"listaMovimentos"`
}

type listarCCExtrato struct {
	omie.PaginacaoResponse
	ListarContasCorrentes []struct {
		NCodCC int64 `json:"nCodCC"`
	} `json:"ListarContasCorrentes"`
}

// minWindowDays é o menor período que tentamos antes de desistir da subdivisão.
const minWindowDays = 1

type ExtratoExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewExtratoExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ExtratoExecutor {
	return &ExtratoExecutor{pool: pool, log: log.With().Str("executor", "extrato").Logger()}
}

func (e *ExtratoExecutor) Nome() string { return "extrato" }

func (e *ExtratoExecutor) ExecutePage(
	ctx context.Context,
	client *omie.Client,
	schema string,
	opts worker.SyncOptions,
	pagina int,
	cfg *omie_config.EndpointConfig,
) (int, error) {
	return 0, fmt.Errorf("ExtratoExecutor.ExecutePage: extrato não suporta sub-jobs por página — usar Execute() com subdivisão binária")
}

func (e *ExtratoExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	_ = rep.Start(ctx, jobID, "extrato")

	// 1. Busca contas correntes
	var ccResp listarCCExtrato
	if err := client.CallPublic(ctx, "/geral/contacorrente/", "ListarContasCorrentes",
		omie.PaginacaoParams{Pagina: 1, RegistrosPorPagina: 200}, &ccResp); err != nil {
		e.log.Warn().Err(err).Msg("extrato: não foi possível listar contas correntes")
		_ = rep.Done(ctx, jobID, "extrato", 0)
		return nil
	}
	if len(ccResp.ListarContasCorrentes) == 0 {
		_ = rep.Done(ctx, jobID, "extrato", 0)
		return nil
	}

	inicio := time.Now()
	fim := time.Now().AddDate(1, 0, 0)

	totalContas := len(ccResp.ListarContasCorrentes)
	total := 0
	for i, cc := range ccResp.ListarContasCorrentes {
		// Limpa provisões anteriores desta conta e empresa e reinicia
		if _, err := e.pool.Exec(ctx,
			fmt.Sprintf("DELETE FROM %s.extrato WHERE empresa_id = $1 AND codigo_conta_corrente = $2", schema),
			opts.EmpresaID, cc.NCodCC,
		); err != nil {
			_ = rep.Fail(ctx, jobID, "extrato", err, nil, "")
			return fmt.Errorf("extrato: erro ao limpar conta %d: %w", cc.NCodCC, err)
		}

		n, err := e.fetchAdaptive(ctx, client, schema, opts.EmpresaID, cc.NCodCC, inicio, fim, cfg)
		if err != nil {
			_ = rep.Fail(ctx, jobID, "extrato", err, client.LastMaskedPayload, string(client.LastResponseMeta))
			return fmt.Errorf("extrato conta %d: %w", cc.NCodCC, err)
		}
		total += n

		// Reporta progresso por conta processada (usando pagina_atual como conta_atual)
		_ = rep.UpdatePage(ctx, jobID, "extrato", i+1, totalContas, total, 0, client.LastMaskedPayload, client.LastResponseMeta)

		e.log.Debug().
			Int64("conta", cc.NCodCC).
			Int("registros", n).
			Msg("extrato: conta sincronizada")
	}

	_ = rep.Done(ctx, jobID, "extrato", total)
	e.log.Info().Str("modulo", "extrato").Int("total", total).Msg("sync concluído")
	return nil
}

// fetchAdaptive busca o extrato de uma conta no período dado.
// Se ocorrer timeout, divide o período ao meio e tenta cada metade recursivamente.
// Limite mínimo: 1 dia (não subdivide mais).
func (e *ExtratoExecutor) fetchAdaptive(
	ctx context.Context,
	client *omie.Client,
	schema string,
	empresaID string,
	nCodCC int64,
	inicio, fim time.Time,
	cfg *omie_config.EndpointConfig,
) (int, error) {
	// Normaliza para datas sem hora
	inicio = truncDay(inicio)
	fim = truncDay(fim)

	windowDays := int(fim.Sub(inicio).Hours()/24) + 1

	params := listarExtratoParams{
		CodigoContaCorrente: nCodCC,
		DataInicial:         inicio.Format("02/01/2006"),
		DataFinal:           fim.Format("02/01/2006"),
	}

	var resp map[string]json.RawMessage
	err := client.CallPublic(ctx, cfg.EndpointPath, cfg.Action, params, &resp)

	if err != nil {
		// Timeout ou erro de rede → tenta subdivisão binária
		if isTimeoutOrRetryable(err) {
			if windowDays <= minWindowDays {
				// Janela mínima atingida — loga e segue sem esse dia
				e.log.Warn().
					Int64("conta", nCodCC).
					Str("data", inicio.Format("02/01/2006")).
					Msg("extrato: timeout em janela de 1 dia, pulando")
				return 0, nil
			}

			// Divide ao meio
			meio := inicio.Add(time.Duration(windowDays/2) * 24 * time.Hour)
			meio = truncDay(meio)

			e.log.Debug().
				Int64("conta", nCodCC).
				Str("de", inicio.Format("02/01/2006")).
				Str("ate", fim.Format("02/01/2006")).
				Int("window_dias", windowDays).
				Msg("extrato: timeout → subdividindo período ao meio")

			// Primeira metade: inicio → meio-1
			n1, err1 := e.fetchAdaptive(ctx, client, schema, empresaID, nCodCC, inicio, meio.AddDate(0, 0, -1), cfg)
			if err1 != nil {
				return n1, err1
			}

			// Segunda metade: meio → fim
			n2, err2 := e.fetchAdaptive(ctx, client, schema, empresaID, nCodCC, meio, fim, cfg)
			if err2 != nil {
				return n1 + n2, err2
			}

			return n1 + n2, nil
		}

		// Erro de negócio (ex: conta sem extrato) → pula silenciosamente
		e.log.Debug().Err(err).Int64("conta", nCodCC).Msg("extrato: erro não retryable, pulando período")
		return 0, nil
	}

	// Extrai dados
	dataRaw, ok := resp[cfg.ArrayField]
	if !ok {
		return 0, nil // Omie às vezes não retorna o campo se estiver vazio
	}

	var movimentos []OmieExtratoMovimento
	if err := json.Unmarshal(dataRaw, &movimentos); err != nil {
		return 0, fmt.Errorf("extrato: erro ao decodificar movimentos: %w", err)
	}

	// Preserva JSON bruto original de cada elemento para a coluna raw
	var movimentosRaw []json.RawMessage
	_ = json.Unmarshal(dataRaw, &movimentosRaw)

	// Sucesso — persiste os registros
	if len(movimentos) == 0 {
		return 0, nil
	}

	for i, mv := range movimentos {
		mv.CodigoContaCorrente = nCodCC
		raw := toJSON(mv)
		if i < len(movimentosRaw) {
			raw = movimentosRaw[i]
		}
		if _, dbErr := e.pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.extrato
				(empresa_id, data_lancamento, valor, tipo_lancamento, codigo_conta_corrente, descricao, raw, synced_at)
			VALUES ($1, NULLIF($2,'')::DATE, $3, 'PROVISAO', $4, $5, $6, NOW())
		`, schema),
			empresaID, mv.DataLancamento, mv.ValorDocumento, nCodCC, mv.Descricao, raw,
		); dbErr != nil {
			return 0, fmt.Errorf("extrato insert [conta=%d de=%s]: %w", nCodCC, params.DataInicial, dbErr)
		}
	}

	return len(movimentos), nil
}

// isTimeoutOrRetryable verifica se o erro é de timeout ou rede (retentável).
func isTimeoutOrRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "i/o timeout")
}

// truncDay retorna a data sem componente de hora.
func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
