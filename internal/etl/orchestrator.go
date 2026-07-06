package etl

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/sync"
	"omie-sync-api/internal/worker"
)

type Orchestrator struct {
	syncRepo sync.Repository
	log      zerolog.Logger
}

func NewOrchestrator(syncRepo sync.Repository, log zerolog.Logger) *Orchestrator {
	return &Orchestrator{
		syncRepo: syncRepo,
		log:      log.With().Str("component", "orchestrator").Logger(),
	}
}

// DiscoverAndCreatePages descobre o total de páginas de cada módulo e cria
// os sub-jobs em sync_job_pages. Extrato é EXPLICITAMENTE excluído.
func (o *Orchestrator) DiscoverAndCreatePages(
	ctx context.Context,
	jobID string,
	client *omie.Client,
	opts worker.SyncOptions,
	configs map[string]*omie_config.EndpointConfig,
) error {
	for modulo, cfg := range configs {
		// 1. Extrato SEMPRE primeiro — nunca criar sub-jobs para extrato
		if modulo == "extrato" {
			o.log.Info().Str("modulo", modulo).Msg("extrato excluído dos sub-jobs — lógica especial")
			continue
		}

		// 2. Módulos inativos
		if !cfg.Ativo {
			continue
		}

		totalPaginas, err := o.probePageCount(ctx, client, cfg, opts)
		if err != nil {
			// Logar e continuar com os outros módulos — não abortar tudo
			o.log.Error().Err(err).Str("modulo", modulo).Msg("falha ao descobrir total de páginas")
			continue
		}

		if totalPaginas == 0 {
			o.log.Info().Str("modulo", modulo).Msg("nenhum registro a sincronizar")
			continue
		}

		for pagina := 1; pagina <= totalPaginas; pagina++ {
			if err := o.syncRepo.InsertJobPage(ctx, jobID, modulo, pagina, totalPaginas); err != nil {
				return fmt.Errorf("orchestrator.DiscoverAndCreatePages [%s pg%d]: %w", modulo, pagina, err)
			}
		}

		o.log.Info().Str("modulo", modulo).Int("paginas", totalPaginas).Msg("sub-jobs criados")
	}
	return nil
}

// probePageCount chama a página 1 do módulo apenas para descobrir o total.
// Não persiste nenhum dado — apenas lê o total_de_paginas da resposta.
func (o *Orchestrator) probePageCount(
	ctx context.Context,
	client *omie.Client,
	cfg *omie_config.EndpointConfig,
	opts worker.SyncOptions,
) (int, error) {
	// Chama página 1 com pageSize=1 apenas para obter o total
	// A resposta deve conter total_de_paginas
	var resp struct {
		TotalDePaginas   int `json:"total_de_paginas"`
		TotalDeRegistros int `json:"total_de_registros"`
	}
	err := client.CallPublic(ctx, cfg.EndpointPath, cfg.Action,
		buildPaginacao(1, 1, opts), &resp)
	if err != nil {
		if omie.IsSemRegistros(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("probePageCount [%s]: %w", cfg.Action, err)
	}
	if resp.TotalDePaginas == 0 {
		return 1, nil // mínimo de 1 página
	}
	return resp.TotalDePaginas, nil
}
