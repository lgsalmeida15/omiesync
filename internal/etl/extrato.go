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

// mesesPorFatia define o fatiamento proativo da janela de 1 ano.
//
// Pedir o ano inteiro numa chamada só funcionava por acidente: contas grandes
// estouravam o tempo, caíam na subdivisão binária e — quando a metade seguinte
// devolvia erro de negócio por volume — o período era descartado em silêncio.
// Uma conta de produção ficou com dados só até 13/11/2026, exatamente a fronteira
// do segundo nível da subdivisão. Fatiar antes torna o resultado determinístico e
// evita o custo de esperar cada timeout até o fim.
const mesesPorFatia = 3

// horizonteAnos é o alcance da busca de provisões, a partir de hoje.
const horizonteAnos = 1

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

	fatias := fatiarPeriodo(time.Now(), horizonteAnos, mesesPorFatia)

	totalContas := len(ccResp.ListarContasCorrentes)
	total := 0
	for i, cc := range ccResp.ListarContasCorrentes {
		// Coleta TODAS as fatias antes de tocar no banco. O DELETE ficava aqui, antes
		// da busca: qualquer falha deixava a conta vazia, sem repor o que existia.
		var coletado []movimentoColetado
		for _, f := range fatias {
			mvs, err := e.fetchAdaptive(ctx, client, cc.NCodCC, f.inicio, f.fim, cfg)
			if err != nil {
				_ = rep.Fail(ctx, jobID, "extrato", err, client.LastMaskedPayload, string(client.LastResponseMeta))
				return fmt.Errorf("extrato conta %d [%s..%s]: %w",
					cc.NCodCC, f.inicio.Format("02/01/2006"), f.fim.Format("02/01/2006"), err)
			}
			coletado = append(coletado, mvs...)
		}

		n, err := e.persistir(ctx, schema, opts.EmpresaID, cc.NCodCC, coletado)
		if err != nil {
			_ = rep.Fail(ctx, jobID, "extrato", err, nil, "")
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

// fatia é um intervalo fechado [inicio, fim].
type fatia struct{ inicio, fim time.Time }

// movimentoColetado carrega o lançamento já decodificado junto do JSON original,
// para persistir tudo de uma vez depois que a conta inteira foi coletada.
type movimentoColetado struct {
	mv         OmieExtratoMovimento
	raw        json.RawMessage
	fluxoCaixa *string
}

// fatiarPeriodo divide [hoje, hoje+anos] em blocos contíguos de `meses`.
//
// Os blocos não se sobrepõem: cada um termina um dia antes do início do
// seguinte, e o último é fechado exatamente no fim do horizonte.
func fatiarPeriodo(agora time.Time, anos, meses int) []fatia {
	inicio := truncDay(agora)
	limite := inicio.AddDate(anos, 0, 0)

	var out []fatia
	for cursor := inicio; cursor.Before(limite); {
		proximo := cursor.AddDate(0, meses, 0)
		fim := proximo.AddDate(0, 0, -1)
		// Na última fatia o fim é o próprio limite: sem isso o horizonte perderia
		// o dia final, já que cada fatia termina uma véspera antes da seguinte.
		if !proximo.Before(limite) {
			fim = limite
		}
		out = append(out, fatia{inicio: cursor, fim: fim})
		cursor = proximo
	}
	return out
}

// ehLinhaDeSaldo identifica as linhas de saldo que o ListarExtrato devolve junto
// dos lançamentos. Vêm com valor zero, sem cSituacao e com descrição fixa — não
// são provisões e só poluem a tabela.
func ehLinhaDeSaldo(mv OmieExtratoMovimento) bool {
	d := strings.ToUpper(strings.TrimSpace(mv.Descricao))
	return d == "SALDO" || d == "SALDO ANTERIOR"
}

// fetchAdaptive busca o extrato de uma conta no período dado e devolve os
// lançamentos, sem gravar.
//
// Qualquer erro dispara a subdivisão binária — não só timeout. O gatilho restrito
// a timeout era a causa da perda de dados: o Omie recusa intervalos com registros
// demais com erro de NEGÓCIO, que caía fora da subdivisão e descartava o período
// inteiro devolvendo (0, nil). Chegando ao piso de 1 dia sem sucesso, o erro é
// propagado e o job falha, em vez de sumir.
func (e *ExtratoExecutor) fetchAdaptive(
	ctx context.Context,
	client *omie.Client,
	nCodCC int64,
	inicio, fim time.Time,
	cfg *omie_config.EndpointConfig,
) ([]movimentoColetado, error) {
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
		// Ausência de registros é resposta legítima, não falha.
		if omie.IsSemRegistros(err) {
			return nil, nil
		}

		if windowDays <= minWindowDays {
			// Piso da subdivisão. Antes daqui saía (0, nil) e o dia desaparecia sem
			// deixar rastro; agora o job falha e o erro fica visível.
			return nil, fmt.Errorf("extrato: falha irredutível em janela de 1 dia (%s): %w",
				inicio.Format("02/01/2006"), err)
		}

		meio := truncDay(inicio.Add(time.Duration(windowDays/2) * 24 * time.Hour))

		e.log.Warn().Err(err).
			Int64("conta", nCodCC).
			Str("de", inicio.Format("02/01/2006")).
			Str("ate", fim.Format("02/01/2006")).
			Int("window_dias", windowDays).
			Msg("extrato: falha no período → subdividindo ao meio")

		// Primeira metade: inicio → meio-1
		m1, err1 := e.fetchAdaptive(ctx, client, nCodCC, inicio, meio.AddDate(0, 0, -1), cfg)
		if err1 != nil {
			return nil, err1
		}

		// Segunda metade: meio → fim
		m2, err2 := e.fetchAdaptive(ctx, client, nCodCC, meio, fim, cfg)
		if err2 != nil {
			return nil, err2
		}

		return append(m1, m2...), nil
	}

	// cFluxoCaixa é irmão de listaMovimentos no envelope da resposta — indica se a
	// conta entra no fluxo de caixa. Esta resposta é a ÚNICA fonte do campo: o
	// cadastro de /geral/contacorrente/ não o traz. Antes era descartado junto com
	// o resto do envelope, e a matvw o procurava em contas_correntes, onde não existe.
	var fluxoCaixa *string
	if v, ok := resp["cFluxoCaixa"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			fluxoCaixa = &s
		}
	}
	// Extrai dados
	dataRaw, ok := resp[cfg.ArrayField]
	if !ok {
		return nil, nil // Omie às vezes não retorna o campo se estiver vazio
	}

	var movimentos []OmieExtratoMovimento
	if err := json.Unmarshal(dataRaw, &movimentos); err != nil {
		return nil, fmt.Errorf("extrato: erro ao decodificar movimentos: %w", err)
	}

	// Preserva JSON bruto original de cada elemento para a coluna raw
	var movimentosRaw []json.RawMessage
	_ = json.Unmarshal(dataRaw, &movimentosRaw)

	out := make([]movimentoColetado, 0, len(movimentos))
	for i, mv := range movimentos {
		if ehLinhaDeSaldo(mv) {
			continue
		}
		mv.CodigoContaCorrente = nCodCC
		raw := toJSON(mv)
		if i < len(movimentosRaw) {
			raw = movimentosRaw[i]
		}
		out = append(out, movimentoColetado{mv: mv, raw: raw, fluxoCaixa: fluxoCaixa})
	}

	return out, nil
}

// persistir troca as provisões da conta numa única transação.
//
// O DELETE só acontece aqui, depois de toda a coleta ter dado certo: antes ele
// rodava antes da busca, e uma falha deixava a conta vazia sem repor nada.
func (e *ExtratoExecutor) persistir(
	ctx context.Context,
	schema, empresaID string,
	nCodCC int64,
	movs []movimentoColetado,
) (int, error) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("extrato: abrir transação: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s.extrato WHERE empresa_id = $1 AND codigo_conta_corrente = $2", schema),
		empresaID, nCodCC,
	); err != nil {
		return 0, fmt.Errorf("extrato: limpar conta %d: %w", nCodCC, err)
	}

	// cFluxoCaixa é irmão de listaMovimentos no envelope da resposta e indica se a
	// conta entra no fluxo de caixa. Esta resposta é a ÚNICA fonte do campo: o
	// cadastro de /geral/contacorrente/ não o traz.
	var fluxoCaixa *string
	for _, m := range movs {
		if m.fluxoCaixa != nil {
			fluxoCaixa = m.fluxoCaixa
			break
		}
	}
	if fluxoCaixa != nil {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf("UPDATE %s.contas_correntes SET fluxo_caixa = $1 WHERE empresa_id = $2 AND codigo_conta_corrente = $3", schema),
			*fluxoCaixa, empresaID, nCodCC,
		); err != nil {
			return 0, fmt.Errorf("extrato: propagar cFluxoCaixa da conta %d: %w", nCodCC, err)
		}
	}

	for _, m := range movs {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.extrato
				(empresa_id, data_lancamento, valor, tipo_lancamento, codigo_conta_corrente, descricao, fluxo_caixa, raw, synced_at)
			VALUES ($1, TO_DATE(NULLIF($2,''), 'DD/MM/YYYY'), $3, 'PROVISAO', $4, $5, $6, $7, NOW())
		`, schema),
			empresaID, m.mv.DataLancamento, m.mv.ValorDocumento, nCodCC, m.mv.Descricao, m.fluxoCaixa, m.raw,
		); err != nil {
			return 0, fmt.Errorf("extrato insert [conta=%d data=%s]: %w", nCodCC, m.mv.DataLancamento, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("extrato: commit conta %d: %w", nCodCC, err)
	}
	return len(movs), nil
}

// truncDay retorna a data sem componente de hora.
func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
