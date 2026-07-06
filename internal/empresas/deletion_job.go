package empresas

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// DeletionJob executa a cada hora e finaliza exclusões cuja carência expirou.
type DeletionJob struct {
	repo   Repository
	log    zerolog.Logger
	ticker *time.Ticker
	done   chan struct{}
}

func NewDeletionJob(repo Repository, log zerolog.Logger) *DeletionJob {
	return &DeletionJob{
		repo: repo,
		log:  log.With().Str("job", "deletion_job").Logger(),
		done: make(chan struct{}),
	}
}

// Start inicia o job em background. Chame Stop() para encerrar.
func (j *DeletionJob) Start() {
	j.ticker = time.NewTicker(1 * time.Hour)
	go func() {
		// Roda imediatamente na inicialização
		j.run()
		for {
			select {
			case <-j.ticker.C:
				j.run()
			case <-j.done:
				j.ticker.Stop()
				return
			}
		}
	}()
	j.log.Info().Msg("deletion job iniciado")
}

func (j *DeletionJob) Stop() {
	close(j.done)
	j.log.Info().Msg("deletion job encerrado")
}

func (j *DeletionJob) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pending, err := j.repo.ListPendingDeletions(ctx)
	if err != nil {
		j.log.Error().Err(err).Msg("erro ao listar pendências de exclusão")
		return
	}

	if len(pending) == 0 {
		return
	}

	j.log.Info().Int("count", len(pending)).Msg("processando exclusões com carência expirada")

	for _, d := range pending {
		if err := j.repo.MarkDeletionExecuted(ctx, d.ID); err != nil {
			j.log.Error().Err(err).Str("empresa_id", d.EmpresaID).Msg("erro ao marcar exclusão executada")
			continue
		}
		j.log.Info().Str("empresa_id", d.EmpresaID).Msg("empresa removida após carência")
	}
}
