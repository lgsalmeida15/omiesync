package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/worker"
)

// saldoPorContasParams é o payload do ListarExtrato quando se quer só o saldo:
// as duas datas iguais (sempre dia 1 de um mês) com cExibirApenasSaldo="S"
// fazem o Omie devolver listaMovimentos vazia e os saldos soltos no envelope
// da resposta (nSaldoAtual, nSaldoAnterior, nSaldoAtualPrevisto, cFluxoCaixa) —
// mesmo endpoint/action do módulo extrato, payload diferente.
type saldoPorContasParams struct {
	CodigoContaCorrente int64  `json:"nCodCC"`
	CCodIntCC           string `json:"cCodIntCC"`
	DataInicial         string `json:"dPeriodoInicial"`
	DataFinal           string `json:"dPeriodoFinal"`
	ExibirApenasSaldo   string `json:"cExibirApenasSaldo"`
}

// mesesBackfillAnos é quantos anos completos ANTES do corrente o backfill
// cobre. 1 = ano corrente + ano anterior, o suficiente para matvw_gerencial_
// ano_corrente e matvw_gerencial_historico.
const mesesBackfillAnos = 1

// mesesResync é quantos meses, a contar do mês corrente (inclusive), são
// SEMPRE repedidos ao Omie a cada ciclo, mesmo já tendo registro — o Omie
// concilia lançamentos bancários com atraso, e um mês fechado há pouco tempo
// ainda pode mudar. Meses mais antigos que isso, uma vez sincronizados, são
// tratados como imutáveis (não geram nova chamada).
const mesesResync = 3

type SaldoPorContasExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewSaldoPorContasExecutor(pool *pgxpool.Pool, log zerolog.Logger) *SaldoPorContasExecutor {
	return &SaldoPorContasExecutor{pool: pool, log: log.With().Str("executor", "saldo_por_contas").Logger()}
}

func (e *SaldoPorContasExecutor) Nome() string { return "saldo_por_contas" }

func (e *SaldoPorContasExecutor) ExecutePage(
	ctx context.Context,
	client *omie.Client,
	schema string,
	opts worker.SyncOptions,
	pagina int,
	cfg *omie_config.EndpointConfig,
) (int, error) {
	return 0, fmt.Errorf("SaldoPorContasExecutor.ExecutePage: saldo_por_contas não suporta sub-jobs por página — a incrementalidade é por conta×mês, não por página Omie")
}

func (e *SaldoPorContasExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	_ = rep.Start(ctx, jobID, "saldo_por_contas")

	var ccResp listarCCExtrato
	if err := client.CallPublic(ctx, "/geral/contacorrente/", "ListarContasCorrentes",
		omie.PaginacaoParams{Pagina: 1, RegistrosPorPagina: 200}, &ccResp); err != nil {
		e.log.Warn().Err(err).Msg("saldo_por_contas: não foi possível listar contas correntes")
		_ = rep.Done(ctx, jobID, "saldo_por_contas", 0)
		return nil
	}
	if len(ccResp.ListarContasCorrentes) == 0 {
		_ = rep.Done(ctx, jobID, "saldo_por_contas", 0)
		return nil
	}

	agora := truncDay(time.Now())
	desde := time.Date(agora.Year()-mesesBackfillAnos, 1, 1, 0, 0, 0, 0, time.UTC)

	totalContas := len(ccResp.ListarContasCorrentes)
	total, tentativas, falhas := 0, 0, 0

	for i, cc := range ccResp.ListarContasCorrentes {
		jaSincronizados, err := e.periodosExistentes(ctx, schema, opts.EmpresaID, cc.NCodCC, desde)
		if err != nil {
			e.log.Warn().Err(err).Int64("conta", cc.NCodCC).
				Msg("saldo_por_contas: falha ao ler períodos já sincronizados; pulando conta neste ciclo")
			continue
		}

		for _, dia1 := range periodosParaSincronizar(agora, jaSincronizados) {
			tentativas++
			n, err := e.sincronizarPeriodo(ctx, client, schema, opts.EmpresaID, cc.NCodCC, dia1, cfg)
			if err != nil {
				falhas++
				// Uma combinação conta×mês ruim não deve travar as outras — ex.: uma
				// conta criada no meio do ano não tem saldo para meses anteriores.
				e.log.Warn().Err(err).
					Int64("conta", cc.NCodCC).
					Str("periodo", dia1.Format("02/01/2006")).
					Msg("saldo_por_contas: falha num período; seguindo para o próximo")
				continue
			}
			total += n
		}

		_ = rep.UpdatePage(ctx, jobID, "saldo_por_contas", i+1, totalContas, total, 0, client.LastMaskedPayload, client.LastResponseMeta)
	}

	// Se TUDO falhou, é sinal de problema sistêmico (ex.: autenticação) — aí sim
	// o job deve ser marcado como falho, em vez de "concluído" com zero gravado.
	if tentativas > 0 && falhas == tentativas {
		err := fmt.Errorf("saldo_por_contas: todas as %d combinações conta×mês falharam — possível erro sistêmico", tentativas)
		_ = rep.Fail(ctx, jobID, "saldo_por_contas", err, client.LastMaskedPayload, string(client.LastResponseMeta))
		return err
	}

	_ = rep.Done(ctx, jobID, "saldo_por_contas", total)
	e.log.Info().Str("modulo", "saldo_por_contas").Int("total", total).Msg("sync concluído")
	return nil
}

// periodosParaSincronizar devolve os "dia 1" (ano anterior completo + ano
// corrente até o mês presente) que ainda precisam ser pedidos ao Omie: os
// últimos mesesResync meses SEMPRE (conciliação bancária tardia), os meses
// mais antigos só se ainda não há registro em jaSincronizados (chave "AAAA-MM").
//
// Pura e sem banco de propósito — é a única lógica de "o que falta buscar"
// desta parte, e testável isoladamente sem Omie nem Postgres.
func periodosParaSincronizar(agora time.Time, jaSincronizados map[string]bool) []time.Time {
	mesCorrente := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.UTC)
	limiteResync := mesCorrente.AddDate(0, -(mesesResync - 1), 0)

	var out []time.Time
	for ano := agora.Year() - mesesBackfillAnos; ano <= agora.Year(); ano++ {
		for mes := 1; mes <= 12; mes++ {
			dia1 := time.Date(ano, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
			if dia1.After(mesCorrente) {
				continue // mês futuro do ano corrente: ainda não aconteceu
			}
			if !dia1.Before(limiteResync) || !jaSincronizados[dia1.Format("2006-01")] {
				out = append(out, dia1)
			}
		}
	}
	return out
}

// periodosExistentes lista, para uma conta, quais "dia 1" já têm registro em
// saldo_por_contas desde o início da janela de backfill.
func (e *SaldoPorContasExecutor) periodosExistentes(ctx context.Context, schema, empresaID string, nCodCC int64, desde time.Time) (map[string]bool, error) {
	rows, err := e.pool.Query(ctx, fmt.Sprintf(
		"SELECT data_referencia FROM %s.saldo_por_contas WHERE empresa_id = $1 AND codigo_conta_corrente = $2 AND data_referencia >= $3",
		schema), empresaID, nCodCC, desde)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out[d.Format("2006-01")] = true
	}
	return out, rows.Err()
}

// sincronizarPeriodo busca e grava o saldo de uma conta no dia 1 de um mês.
// Ausência de registros (conta ainda não existia naquele mês) não é falha.
func (e *SaldoPorContasExecutor) sincronizarPeriodo(
	ctx context.Context,
	client *omie.Client,
	schema, empresaID string,
	nCodCC int64,
	dia1 time.Time,
	cfg *omie_config.EndpointConfig,
) (int, error) {
	dataStr := dia1.Format("02/01/2006")
	params := saldoPorContasParams{
		CodigoContaCorrente: nCodCC,
		CCodIntCC:           "",
		DataInicial:         dataStr,
		DataFinal:           dataStr,
		ExibirApenasSaldo:   "S",
	}

	var resp map[string]json.RawMessage
	if err := client.CallPublic(ctx, cfg.EndpointPath, cfg.Action, params, &resp); err != nil {
		if omie.IsSemRegistros(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("saldo_por_contas conta %d [%s]: %w", nCodCC, dataStr, err)
	}

	getFloat := func(chave string) float64 {
		v, ok := resp[chave]
		if !ok {
			return 0
		}
		var f float64
		_ = json.Unmarshal(v, &f)
		return f
	}

	// Reconstrói o envelope inteiro para a coluna raw — preserva campos que não
	// viraram coluna própria (cDescricao, nSaldoConciliado, nSaldoDisponivel etc.),
	// igual ao padrão de extrato.go para cFluxoCaixa.
	rawBytes, err := json.Marshal(resp)
	if err != nil {
		rawBytes = nil
	}

	if err := e.persistirPeriodo(ctx, schema, empresaID, nCodCC, dia1,
		getFloat("nSaldoAnterior"), getFloat("nSaldoAtual"), getFloat("nSaldoAtualPrevisto"), rawBytes); err != nil {
		return 0, fmt.Errorf("saldo_por_contas: gravar conta %d [%s]: %w", nCodCC, dataStr, err)
	}
	return 1, nil
}

// persistirPeriodo grava por UPSERT, não DELETE+INSERT: cada linha é um fato
// histórico independente por mês, diferente de extrato (que substitui uma
// lista inteira a cada sync).
func (e *SaldoPorContasExecutor) persistirPeriodo(
	ctx context.Context,
	schema, empresaID string,
	nCodCC int64,
	dia1 time.Time,
	saldoAnterior, saldoAtual, saldoAtualPrevisto float64,
	raw []byte,
) error {
	_, err := e.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.saldo_por_contas
			(empresa_id, codigo_conta_corrente, data_referencia, saldo_anterior, saldo_atual, saldo_atual_previsto, raw, synced_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (empresa_id, codigo_conta_corrente, data_referencia) DO UPDATE SET
			saldo_anterior       = EXCLUDED.saldo_anterior,
			saldo_atual          = EXCLUDED.saldo_atual,
			saldo_atual_previsto = EXCLUDED.saldo_atual_previsto,
			raw                  = EXCLUDED.raw,
			synced_at            = NOW()
	`, schema),
		empresaID, nCodCC, dia1, saldoAnterior, saldoAtual, saldoAtualPrevisto, raw,
	)
	return err
}
