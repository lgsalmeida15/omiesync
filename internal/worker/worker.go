package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/db"
	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	syncsvc "omie-sync-api/internal/sync"
	"omie-sync-api/internal/webhooks"
)

// EmpresaCredentials fornece credenciais de uma empresa para o worker.
type EmpresaCredentials struct {
	ID        string
	GrupoID   string
	AppKey    string
	AppSecret string
	Schema    string
}

// EmpresaFetcher busca credenciais e gerencia status de empresas.
type EmpresaFetcher interface {
	GetActiveCredentials(ctx context.Context, empresaID string) (*EmpresaCredentials, error)
	PausarEmpresa(ctx context.Context, empresaID string) error
}

// SyncOptions parâmetros passados ao executor para controle do tipo de sync.
type SyncOptions struct {
	// UltimoSyncAt — data de corte no formato "DD/MM/YYYY" (hoje em delta syncs).
	// Executors que suportam delta usam como FiltrarPorDataDe no Omie.
	UltimoSyncAt string

	// Full — true em tipo="full": ignora UltimoSyncAt, busca tudo.
	Full bool

	// IgnorarDelta — executor deve buscar TODOS os registros mesmo em sync delta.
	// Usado por: movimentos_financeiros, contas_correntes.
	// Razão: movimentos têm UPSERT por ID único, sempre completo garante consistência.
	// Contas correntes são dados mestres com poucos registros.
	IgnorarDelta bool

	// EmpresaID — UUID da empresa dona dos registros.
	// Obrigatório para isolamento multi-empresa dentro do schema de grupo.
	EmpresaID string
}

// Executor processa um tipo de sync (ex: clientes, contas_pagar).
type Executor interface {
	Nome() string
	Execute(ctx context.Context, client *omie.Client, schemaName string, opts SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error
	// ExecutePage processa uma página específica e retorna o número de registros gravados.
	// Retorna (0, nil) se a página não tiver registros.
	ExecutePage(ctx context.Context, client *omie.Client, schema string, opts SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error)
}

// Worker executa jobs de sync da fila.
type Worker struct {
	syncRepo   syncsvc.Repository
	fetcher    EmpresaFetcher
	executors  []Executor
	dispatcher webhooks.Dispatcher
	reporter   progress.Reporter
	omieConfig omie_config.Service
	hub        *syncsvc.SSEHub
	pool       *pgxpool.Pool
	log        zerolog.Logger

	// Coalescência de refresh por schema. As views gerenciais são do GRUPO, mas o
	// sync é por empresa — um grupo com 5 empresas disparava 5 refreshes completos
	// da mesma view, serializados pelo lock exclusivo do REFRESH. Observado em
	// produção: 5 refreshes empilhados, o primeiro rodando há 26 minutos e os
	// outros em espera, com o dashboard inacessível o tempo todo.
	refreshMu    sync.Mutex
	refreshAtivo map[string]*refreshState
}

type refreshState struct {
	// pendente sinaliza que chegou pedido novo enquanto um refresh rodava; ao
	// terminar, roda mais uma vez para incorporar os dados que chegaram depois.
	pendente bool
	// full acumula: se qualquer pedido da janela pediu histórico, o refresh final
	// também atualiza a histórica.
	full bool
}

func NewWorker(
	syncRepo syncsvc.Repository,
	fetcher EmpresaFetcher,
	executors []Executor,
	dispatcher webhooks.Dispatcher,
	reporter progress.Reporter,
	omieConfig omie_config.Service,
	hub *syncsvc.SSEHub,
	pool *pgxpool.Pool,
	log zerolog.Logger,
) *Worker {
	return &Worker{
		syncRepo:   syncRepo,
		fetcher:    fetcher,
		executors:  executors,
		dispatcher: dispatcher,
		reporter:   reporter,
		omieConfig: omieConfig,
		hub:        hub,
		pool:       pool,
		log:        log.With().Str("component", "worker").Logger(),

		refreshAtivo: make(map[string]*refreshState),
	}
}

// agendarRefresh atualiza as views gerenciais do schema, garantindo no máximo um
// REFRESH em andamento por schema.
//
// O REFRESH (sem CONCURRENTLY) toma lock exclusivo e bloqueia toda leitura da view.
// Como as views são do grupo e o sync é por empresa, sem coalescência um grupo com
// N empresas enfileira N refreshes idênticos, e o dashboard fica indisponível
// durante a soma de todos eles.
//
// Se chegar pedido enquanto um refresh roda, marca pendente em vez de enfileirar:
// ao terminar, executa mais uma vez — o suficiente para incorporar o que chegou no
// intervalo, independente de quantos pedidos tenham ocorrido.
func (w *Worker) agendarRefresh(schema string, full bool) {
	w.refreshMu.Lock()
	if st, rodando := w.refreshAtivo[schema]; rodando {
		st.pendente = true
		st.full = st.full || full
		w.refreshMu.Unlock()
		w.log.Debug().Str("schema", schema).Msg("worker: refresh já em andamento, coalescido")
		return
	}
	w.refreshAtivo[schema] = &refreshState{full: full}
	w.refreshMu.Unlock()

	go func() {
		for {
			w.refreshMu.Lock()
			st := w.refreshAtivo[schema]
			executarFull := st.full
			st.pendente = false
			st.full = false
			w.refreshMu.Unlock()

			w.executarRefresh(schema, executarFull)

			w.refreshMu.Lock()
			if !w.refreshAtivo[schema].pendente {
				delete(w.refreshAtivo, schema)
				w.refreshMu.Unlock()
				return
			}
			w.refreshMu.Unlock()
		}
	}()
}

func (w *Worker) executarRefresh(schema string, full bool) {
	w.refreshView(schema, "matvw_gerencial_ano_corrente", 30*time.Minute)
	if full {
		w.refreshView(schema, "matvw_gerencial_historico", 2*time.Hour)
	}
}

// refreshView atualiza uma view materializada preferindo CONCURRENTLY, que não toma
// lock exclusivo — os leitores continuam enxergando o conteúdo anterior enquanto a
// nova versão é construída.
//
// Sem isso o dashboard ficava indisponível durante todo o refresh. Com 5 empresas
// sincronizando de hora em hora e jobs de até 26 minutos, a colisão era rotina.
//
// CONCURRENTLY exige que a view já esteja populada e tenha índice único (mv_id). Na
// primeira carga, ou logo após um re-provisionamento que recria WITH NO DATA, é
// preciso um refresh normal — e é justamente ele que produz o "dados ainda não
// disponíveis" até concluir.
func (w *Worker) refreshView(schema, view string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	safe := pgx.Identifier{schema}.Sanitize()
	alvo := fmt.Sprintf("%s.%s", safe, pgx.Identifier{view}.Sanitize())

	var populada bool
	if err := w.pool.QueryRow(ctx, `
		SELECT c.relispopulated
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`, schema, view).Scan(&populada); err != nil {
		w.log.Warn().Err(err).Str("schema", schema).Str("view", view).
			Msg("worker: não foi possível verificar se a view está populada; usando refresh bloqueante")
		populada = false
	}

	modo := "CONCURRENTLY "
	if !populada {
		modo = ""
	}

	inicio := time.Now()
	_, err := w.pool.Exec(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW %s%s", modo, alvo))

	// CONCURRENTLY pode ser recusado se o índice único ainda não existir (schema de
	// versão anterior). Cai para o modo bloqueante em vez de deixar a view desatualizada.
	if err != nil && populada {
		w.log.Warn().Err(err).Str("schema", schema).Str("view", view).
			Msg("worker: refresh concorrente falhou; tentando modo bloqueante")
		_, err = w.pool.Exec(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", alvo))
	}

	if err != nil {
		w.log.Warn().Err(err).Str("schema", schema).Str("view", view).Msg("worker: refresh falhou")
		return
	}
	w.log.Info().
		Str("schema", schema).Str("view", view).
		Bool("concorrente", populada).
		Dur("duracao", time.Since(inicio)).
		Msg("worker: view gerencial atualizada")
}

// ProcessJob executa um job específico pelo ID.
func (w *Worker) ProcessJob(ctx context.Context, jobID string) error {
	job, err := w.syncRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("worker.ProcessJob buscar job: %w", err)
	}

	creds, err := w.fetcher.GetActiveCredentials(ctx, job.EmpresaID)
	if err != nil {
		return fmt.Errorf("worker.ProcessJob buscar credenciais: %w", err)
	}

	return w.execute(ctx, job, creds)
}

func (w *Worker) execute(ctx context.Context, job *syncsvc.SyncJob, creds *EmpresaCredentials) error {
	now := time.Now()

	// Configura hub no reporter para esta execução
	w.reporter.SetHub(w.hub, job.EmpresaID)

	// Marca job como rodando
	if _, err := w.syncRepo.UpdateJobStatus(ctx, job.ID, "rodando", "", &now, nil); err != nil {
		return fmt.Errorf("worker.execute marcar rodando: %w", err)
	}

	// Provisiona schema apenas quando a versão está desatualizada — evita DDL lock contention
	provisioner := db.NewProvisioner(w.pool)
	reprovisionado := false
	if provisioner.NeedsProvisioning(ctx, creds.Schema) {
		if err := provisioner.ProvisionSchema(ctx, creds.Schema); err != nil {
			w.log.Warn().Err(err).Str("schema", creds.Schema).Msg("worker: re-provision schema falhou (continuando)")
		} else {
			// Os auto-upgrades podem dropar e recriar as views gerenciais WITH NO DATA.
			// Sem forçar o refresh da histórica aqui, ela ficaria vazia até o próximo
			// sync full — e consultar matvw não populada é erro (SQLSTATE 55000), não
			// resultado vazio. Foi o que quebrou o dashboard em anos anteriores.
			reprovisionado = true
			w.log.Info().Str("schema", creds.Schema).Msg("worker: schema re-provisionado, views serão repopuladas")
		}
	}

	w.hub.Publish(job.EmpresaID, syncsvc.SSEEvent{
		Type: "job.iniciado",
		Data: job,
	})

	w.log.Info().
		Str("job_id", job.ID).
		Str("empresa_id", job.EmpresaID).
		Str("tipo", job.Tipo).
		Msg("iniciando sync")

	client := omie.NewClient(creds.AppKey, creds.AppSecret)
	var erros []string

	// Inicializa progresso para todos os executors
	executorNames := make([]string, len(w.executors))
	for i, exec := range w.executors {
		executorNames[i] = exec.Nome()
	}
	if err := w.reporter.Init(ctx, job.ID, executorNames); err != nil {
		w.log.Error().Err(err).Str("job_id", job.ID).Msg("erro ao inicializar progresso")
		w.finalizarJob(ctx, job.ID, creds.GrupoID, job.EmpresaID, "erro", "falha ao inicializar rastreador de progresso", now)
		return fmt.Errorf("worker.execute inicializar progresso: %w", err)
	}

	// Monta opções de sync:
	// - Full (tipo="full"): busca TUDO, sem filtro de data
	// - Delta (automático/manual): filtra por HOJE
	//   O Omie usa data_alteracao (granularidade dia, não hora).
	//   Filtrar por hoje garante captura de alterações feitas em qualquer
	//   horário do dia, independente do horário do último sync.
	opts := SyncOptions{Full: job.Tipo == "full", EmpresaID: creds.ID}
	if !opts.Full {
		opts.UltimoSyncAt = time.Now().Format("02/01/2006")
	}

	// Executa cada módulo de sync
	disabledExecutors, _ := w.syncRepo.GetEnabledExecutors(ctx, job.EmpresaID)

	for _, exec := range w.executors {
		nome := exec.Nome()

		// 1. Pula se executor desabilitado para esta empresa
		if disabledExecutors[nome] {
			_ = w.reporter.Skip(ctx, job.ID, nome, "desabilitado para esta empresa")
			w.log.Debug().Str("executor", nome).Msg("executor desabilitado para esta empresa, pulando")
			continue
		}

		// 2. Pula se job tem executor específico e este não é ele
		if job.Executor != "" && job.Executor != nome {
			_ = w.reporter.Skip(ctx, job.ID, nome, "não selecionado neste job")
			w.log.Debug().Str("executor", nome).Msg("executor não selecionado para este job seletivo, pulando")
			continue
		}

		cfg, err := w.omieConfig.GetCached(ctx, nome)
		if err != nil {
			w.log.Error().Err(err).Str("executor", exec.Nome()).Msg("erro ao buscar configuração do módulo")
			erros = append(erros, fmt.Sprintf("%s: erro config", exec.Nome()))
			continue
		}

		if !cfg.Ativo {
			w.log.Debug().Str("executor", exec.Nome()).Msg("executor inativo, pulando")
			continue
		}

		// Propaga cfg.IgnorarDelta → opts para que buildPaginacao não aplique filtro de data
		// em módulos que a API do Omie não suporta (ex: categorias, departamentos).
		execOpts := opts
		if cfg.IgnorarDelta {
			execOpts.IgnorarDelta = true
		}

		stop := false
		func() {
			const jobTimeout = 2 * time.Hour
			execCtx, cancelExec := context.WithTimeout(ctx, jobTimeout)
			defer cancelExec()

			if err := exec.Execute(execCtx, client, creds.Schema, execOpts, job.ID, w.reporter, cfg); err != nil {
				erroMsg := fmt.Sprintf("%s: %v", exec.Nome(), err)
				erros = append(erros, erroMsg)
				w.log.Error().Err(err).Str("executor", exec.Nome()).Msg("erro no executor")

				// Credencial inválida — pausa a empresa no banco + webhook
				if omie.IsCredencialInvalida(err) {
					if pauseErr := w.fetcher.PausarEmpresa(ctx, job.EmpresaID); pauseErr != nil {
						w.log.Error().Err(pauseErr).Str("empresa_id", job.EmpresaID).Msg("erro ao pausar empresa")
					}
					w.finalizarJob(ctx, job.ID, creds.GrupoID, job.EmpresaID, "erro",
						"credencial inválida — empresa pausada automaticamente", now)
					w.dispatcher.Dispatch(creds.GrupoID, webhooks.Event{
						Tipo:       webhooks.EventEmpresaPausada,
						GrupoID:    creds.GrupoID,
						EmpresaID:  job.EmpresaID,
						OcorridoAt: time.Now(),
					})
					stop = true
					return
				}
			}
		}()

		if stop {
			return nil
		}
	}

	// Atualiza controle após run bem-sucedido
	_ = w.syncRepo.UpdateControlAfterRun(ctx, job.EmpresaID, job.Tipo)

	if len(erros) > 0 {
		msg := fmt.Sprintf("%d erros: %v", len(erros), erros)
		w.finalizarJob(ctx, job.ID, creds.GrupoID, job.EmpresaID, "erro", msg, now)
	} else {
		w.finalizarJob(ctx, job.ID, creds.GrupoID, job.EmpresaID, "concluido", "", now)

		// Refresh assíncrono das views gerenciais após sync bem-sucedido.
		// Incremental: apenas ano_corrente. Full ou re-provisionamento: as duas.
		if w.pool != nil {
			// Re-provisionamento entra aqui: se as views foram recriadas WITH NO DATA,
			// a histórica precisa ser repopulada mesmo num sync incremental.
			w.agendarRefresh(creds.Schema, job.Tipo == "full" || reprovisionado)
		}
	}

	return nil
}

func (w *Worker) finalizarJob(ctx context.Context, jobID, grupoID, empresaID, status, erroMsg string, iniciado time.Time) {
	fim := time.Now()
	job, err := w.syncRepo.UpdateJobStatus(ctx, jobID, status, erroMsg, &iniciado, &fim)
	if err != nil {
		w.log.Error().Err(err).Str("job_id", jobID).Msg("erro ao finalizar job")
	}

	// Notifica via SSE
	evtType := "job.concluido"
	if status == "erro" {
		evtType = "job.erro"
	}
	if job != nil {
		w.hub.Publish(empresaID, syncsvc.SSEEvent{
			Type: evtType,
			Data: job,
		})
	}

	evento := webhooks.EventSyncConcluido
	if status == "erro" {
		evento = webhooks.EventSyncFalhou
	}

	w.dispatcher.Dispatch(grupoID, webhooks.Event{
		Tipo:       evento,
		GrupoID:    grupoID,
		EmpresaID:  empresaID,
		OcorridoAt: fim,
	})

	w.log.Info().
		Str("job_id", jobID).
		Str("status", status).
		Dur("duracao", fim.Sub(iniciado)).
		Msg("job finalizado")
}
