package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	syncrepo "omie-sync-api/internal/sync"
	"omie-sync-api/internal/worker"
)

const maxConcurrentPages = 5

// backoffDuration retorna o tempo de espera baseado no número de tentativas já feitas.
func backoffDuration(tentativas int) time.Duration {
	switch tentativas {
	case 1:
		return 30 * time.Second
	case 2:
		return 2 * time.Minute
	default:
		return 5 * time.Minute
	}
}

type PageWorker struct {
	syncRepo  syncrepo.Repository
	executors map[string]worker.Executor // mapa modulo → executor ETL
	configs   map[string]*omie_config.EndpointConfig
	log       zerolog.Logger
}

func NewPageWorker(
	syncRepo syncrepo.Repository,
	executors map[string]worker.Executor,
	configs map[string]*omie_config.EndpointConfig,
	log zerolog.Logger,
) *PageWorker {
	return &PageWorker{
		syncRepo:  syncRepo,
		executors: executors,
		configs:   configs,
		log:       log.With().Str("component", "page_worker").Logger(),
	}
}

// ProcessJob processa todas as páginas pendentes de um job até não restar nenhuma.
func (pw *PageWorker) ProcessJob(ctx context.Context, jobID string, client *omie.Client, opts worker.SyncOptions, schema string) error {
	sem := make(chan struct{}, maxConcurrentPages)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for {
		// Verificar se contexto foi cancelado
		select {
		case <-ctx.Done():
			return fmt.Errorf("page_worker.ProcessJob: contexto cancelado: %w", ctx.Err())
		default:
		}

		pending, err := pw.syncRepo.GetPendingPages(ctx, jobID, maxConcurrentPages)
		if err != nil {
			return fmt.Errorf("page_worker.ProcessJob: %w", err)
		}
		if len(pending) == 0 {
			count, _ := pw.syncRepo.CountPendingPages(ctx, jobID)
			if count == 0 {
				break // todas concluídas ou na DLQ
			}
			// Há páginas em backoff — aguardar antes de checar novamente
			time.Sleep(5 * time.Second)
			continue
		}

		for _, page := range pending {
			wg.Add(1)
			sem <- struct{}{}

			go func(p syncrepo.JobPage) {
				defer wg.Done()
				defer func() { <-sem }()

				if err := pw.processPage(ctx, p, client, opts, schema); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}(page)
		}

		wg.Wait()
	}

	return firstErr
}

func (pw *PageWorker) processPage(
	ctx context.Context,
	page syncrepo.JobPage,
	client *omie.Client,
	opts worker.SyncOptions,
	schema string,
) error {
	log := pw.log.With().Str("modulo", page.Modulo).Int("pagina", page.Pagina).Logger()

	// Claim atômico — evita processamento duplo
	claimed, err := pw.syncRepo.ClaimPageForProcessing(ctx, page.ID)
	if err != nil || claimed == nil {
		return nil // outra goroutine já pegou esta página
	}

	// Guard de segurança: extrato jamais deve chegar aqui
	if page.Modulo == "extrato" {
		pw.log.Error().Msg("BUG CRÍTICO: extrato chegou ao PageWorker — isso nunca deve acontecer")
		_ = pw.syncRepo.MarkPageErro(ctx, page.ID,
			"BUG: extrato não deve usar sub-jobs por página", time.Now().Add(24*time.Hour))
		return nil
	}

	cfg, ok := pw.configs[page.Modulo]
	if !ok {
		_ = pw.syncRepo.MarkPageErro(ctx, page.ID, "executor não encontrado para módulo: "+page.Modulo, time.Now())
		return nil
	}

	exec, ok := pw.executors[page.Modulo]
	if !ok {
		_ = pw.syncRepo.MarkPageErro(ctx, page.ID, "executor não registrado: "+page.Modulo, time.Now())
		return nil
	}

	// Executar a página específica
	registros, err := exec.ExecutePage(ctx, client, schema, opts, page.Pagina, cfg)
	if err != nil {
		backoff := backoffDuration(claimed.Tentativas)
		proximoRetry := time.Now().Add(backoff)

		log.Warn().Err(err).Int("tentativa", claimed.Tentativas).
			Dur("proximo_retry_em", backoff).Msg("falha ao processar página")

		if claimed.Tentativas >= claimed.MaxTentativas {
			// Esgotou as tentativas → entra na DLQ (status=erro permanente)
			_ = pw.syncRepo.MarkPageErro(ctx, page.ID,
				fmt.Sprintf("[DLQ] %s", err.Error()),
				time.Time{}) // sem próximo retry
			return nil
		}

		_ = pw.syncRepo.MarkPageErro(ctx, page.ID, err.Error(), proximoRetry)
		return nil
	}

	if err := pw.syncRepo.MarkPageConcluido(ctx, page.ID, registros); err != nil {
		return fmt.Errorf("page_worker.processPage.MarkPageConcluido: %w", err)
	}

	log.Info().Int("registros", registros).Msg("página processada com sucesso")
	return nil
}
