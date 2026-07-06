package etl

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/worker"
)

// alwaysFullWrapper envolve um Executor e força IgnorarDelta=true em todas as execuções.
// Usado por módulos que sempre devem buscar tudo: movimentos_financeiros, contas_correntes.
type alwaysFullWrapper struct {
	inner worker.Executor
}

func (w *alwaysFullWrapper) Nome() string { return w.inner.Nome() }
func (w *alwaysFullWrapper) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	opts.IgnorarDelta = true
	return w.inner.Execute(ctx, client, schema, opts, jobID, rep, cfg)
}

func (w *alwaysFullWrapper) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	opts.IgnorarDelta = true
	return w.inner.ExecutePage(ctx, client, schema, opts, pagina, cfg)
}

// NewAllExecutors cria a lista completa de executors na ordem recomendada de sync.
//
// Regras de delta:
//   - Filtram por HOJE (data_alteracao):
//     clientes, categorias, contas_pagar, contas_receber, ordens_servico, projetos
//   - Sempre buscam TUDO (IgnorarDelta=true):
//     contas_correntes — dados mestres, poucos registros
//     movimentos_financeiros — UPSERT por ID único, sempre completo garante consistência
//   - Lógica própria (sem buildPaginacao):
//     extrato — sempre hoje +1 ano (provisão futura), subdivisão binária adaptativa
func NewAllExecutors(pool *pgxpool.Pool, log zerolog.Logger) []worker.Executor {
	return []worker.Executor{
		NewCategoriasExecutor(pool, log),                                        // delta: hoje
		NewDepartamentosExecutor(pool, log),                                     // delta: hoje
		&alwaysFullWrapper{NewContasCorrentesExecutor(pool, log)},               // sempre tudo
		NewClientesExecutor(pool, log),                                          // delta: hoje
		NewContasPagarExecutor(pool, log),                                       // delta: hoje
		NewContasReceberExecutor(pool, log),                                     // delta: hoje
		NewMovimentosFinanceirosExecutor(pool, log),                             // datas via dDtIncDe/dDtAltDe
		NewExtratoExecutor(pool, log),                                           // hoje +365d (lógica própria)
		NewOrdensServicoExecutor(pool, log),                                     // delta: hoje
		NewProjetosExecutor(pool, log),                                          // delta: hoje
	}
}
