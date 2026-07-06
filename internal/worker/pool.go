package worker

import (
	"context"

	"github.com/rs/zerolog"
)

// WorkerPool limita quantos jobs rodam simultaneamente via semáforo (canal com capacidade N).
// Cada Submit lança uma goroutine, mas ela só executa fn() após adquirir um slot.
// Quando fn() termina, o slot é liberado automaticamente e a próxima goroutine da fila entra.
type WorkerPool struct {
	sem chan struct{}
	log zerolog.Logger
}

func NewWorkerPool(maxConcurrent int, log zerolog.Logger) *WorkerPool {
	return &WorkerPool{
		sem: make(chan struct{}, maxConcurrent),
		log: log.With().Str("component", "worker_pool").Logger(),
	}
}

// Submit envia fn para execução. Bloqueia até haver slot disponível, respeitando ctx.
func (p *WorkerPool) Submit(ctx context.Context, jobID string, fn func()) {
	go func() {
		select {
		case p.sem <- struct{}{}:
		case <-ctx.Done():
			p.log.Warn().Str("job_id", jobID).Msg("worker_pool: job cancelado antes de iniciar (ctx encerrado)")
			return
		}
		defer func() { <-p.sem }()
		fn()
	}()
}
