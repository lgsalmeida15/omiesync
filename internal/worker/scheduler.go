package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	syncsvc "omie-sync-api/internal/sync"
)

// Scheduler verifica a cada minuto quais empresas precisam de sync
// e cria jobs automaticamente para o worker processar.
type Scheduler struct {
	pool       *pgxpool.Pool
	syncRepo   syncsvc.Repository
	worker     *Worker
	workerPool *WorkerPool
	log        zerolog.Logger
	done       chan struct{}
}

func NewScheduler(pool *pgxpool.Pool, syncRepo syncsvc.Repository, worker *Worker, maxConcurrent int, log zerolog.Logger) *Scheduler {
	return &Scheduler{
		pool:       pool,
		syncRepo:   syncRepo,
		worker:     worker,
		workerPool: NewWorkerPool(maxConcurrent, log),
		log:        log.With().Str("component", "scheduler").Logger(),
		done:       make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		s.tick() // roda imediatamente ao iniciar
		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-s.done:
				ticker.Stop()
				return
			}
		}
	}()
	s.log.Info().Msg("scheduler iniciado")
}

func (s *Scheduler) Stop() {
	close(s.done)
	s.log.Info().Msg("scheduler encerrado")
}

// Pool retorna o WorkerPool para que outros componentes (ex: ForcarSync) passem pelo semáforo.
func (s *Scheduler) Pool() *WorkerPool {
	return s.workerPool
}

func (s *Scheduler) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()

	// 1. Incremental syncs
	s.processIncremental(ctx)

	// 2. Full syncs
	s.processFull(ctx)
}

func (s *Scheduler) processIncremental(ctx context.Context) {
	empresas, err := s.getEmpresasDueIncremental(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("scheduler: erro ao buscar empresas para incremental")
		return
	}

	for _, empresaID := range empresas {
		// 1. Avança o schedule IMEDIATAMENTE (comportamento cron puro)
		if err := s.syncRepo.AdvanceScheduleOnDispatch(ctx, empresaID, "automatico"); err != nil {
			s.log.Error().Err(err).Str("empresa_id", empresaID).Msg("scheduler: erro ao avançar schedule incremental")
			continue
		}

		// 2. Tratamento de drift: se o próximo sync ainda estiver no passado, avança até alcançar o presente.
		// Isso evita "empilhamento" de jobs se o servidor ficou offline por muito tempo.
		for {
			ctrl, err := s.syncRepo.GetControl(ctx, empresaID)
			if err != nil || ctrl.ProximoSyncAt == nil || ctrl.ProximoSyncAt.After(time.Now()) {
				break
			}
			s.log.Debug().Str("empresa_id", empresaID).Time("proximo", *ctrl.ProximoSyncAt).Msg("scheduler: corrigindo drift incremental")
			_ = s.syncRepo.AdvanceScheduleOnDispatch(ctx, empresaID, "automatico")
		}

		// 3. Cria o job
		job, err := s.syncRepo.InsertJob(ctx, empresaID, "automatico", "")
		if err != nil {
			s.log.Error().Err(err).Str("empresa_id", empresaID).Msg("scheduler: erro ao criar job incremental")
			continue
		}

		s.log.Info().Str("job_id", job.ID).Str("empresa_id", empresaID).Msg("scheduler: job incremental criado")
		s.startWorker(job.ID)
	}
}

func (s *Scheduler) processFull(ctx context.Context) {
	empresas, err := s.getEmpresasDueFull(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("scheduler: erro ao buscar empresas para full")
		return
	}

	for _, empresaID := range empresas {
		// 1. Avança o schedule IMEDIATAMENTE
		if err := s.syncRepo.AdvanceScheduleOnDispatch(ctx, empresaID, "full"); err != nil {
			s.log.Error().Err(err).Str("empresa_id", empresaID).Msg("scheduler: erro ao avançar schedule full")
			continue
		}

		// 2. Tratamento de drift full
		for {
			ctrl, err := s.syncRepo.GetControl(ctx, empresaID)
			if err != nil || ctrl.ProximoFullSyncAt == nil || ctrl.ProximoFullSyncAt.After(time.Now()) {
				break
			}
			s.log.Debug().Str("empresa_id", empresaID).Time("proximo_full", *ctrl.ProximoFullSyncAt).Msg("scheduler: corrigindo drift full")
			_ = s.syncRepo.AdvanceScheduleOnDispatch(ctx, empresaID, "full")
		}

		// 3. Cria o job
		job, err := s.syncRepo.InsertJob(ctx, empresaID, "full", "")
		if err != nil {
			s.log.Error().Err(err).Str("empresa_id", empresaID).Msg("scheduler: erro ao criar job full")
			continue
		}

		s.log.Info().Str("job_id", job.ID).Str("empresa_id", empresaID).Msg("scheduler: job full criado")
		s.startWorker(job.ID)
	}
}

func (s *Scheduler) startWorker(jobID string) {
	s.workerPool.Submit(context.Background(), jobID, func() {
		jobCtx, jobCancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer jobCancel()
		if err := s.worker.ProcessJob(jobCtx, jobID); err != nil {
			s.log.Error().Err(err).Str("job_id", jobID).Msg("scheduler: erro ao processar job")
		}
	})
}

func (s *Scheduler) getEmpresasDueIncremental(ctx context.Context) ([]string, error) {
	return s.getEmpresasDue(ctx, `
		SELECT sc.empresa_id::text
		FROM _etl.sync_control sc
		JOIN _etl.empresas e ON e.id = sc.empresa_id
		WHERE sc.ativo = true
		  AND sc.proximo_sync_at <= NOW()
		  AND e.status = 'ativa'
		  AND e.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM _etl.sync_jobs sj
			WHERE sj.empresa_id = sc.empresa_id
			  AND sj.status IN ('pendente', 'rodando')
		  )
	`)
}

func (s *Scheduler) getEmpresasDueFull(ctx context.Context) ([]string, error) {
	return s.getEmpresasDue(ctx, `
		SELECT sc.empresa_id::text
		FROM _etl.sync_control sc
		JOIN _etl.empresas e ON e.id = sc.empresa_id
		WHERE sc.ativo = true
		  AND sc.proximo_full_sync_at <= NOW()
		  AND e.status = 'ativa'
		  AND e.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM _etl.sync_jobs sj
			WHERE sj.empresa_id = sc.empresa_id
			  AND sj.status IN ('pendente', 'rodando')
		  )
	`)
}

func (s *Scheduler) getEmpresasDue(ctx context.Context, query string) ([]string, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

