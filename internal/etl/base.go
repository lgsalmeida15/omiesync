package etl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/etl/progress"
	"omie-sync-api/internal/omie"
	"omie-sync-api/internal/omie_config"
	"omie-sync-api/internal/worker"
)

// OmieResponse é uma estrutura genérica para capturar paginação e o campo de dados dinâmico.
type OmieResponse struct {
	Pagina           int `json:"pagina"`
	TotalDePaginas   int `json:"total_de_paginas"`
	Registros        int `json:"registros"`
	TotalDeRegistros int `json:"total_de_registros"`
}

// pageFunc é a assinatura de uma função que busca uma página de dados do Omie.
// Retorna os itens tipados, os itens como JSON bruto original (um por elemento) e o total de páginas.
type pageFunc[T any] func(ctx context.Context, client *omie.Client, pagina, pageSize int) ([]T, []json.RawMessage, int, error)

// upsertFunc é a assinatura de uma função que persiste um batch no schema do tenant.
// empresaID identifica a empresa dona dos registros (isolamento multi-empresa no schema).
// raws contém o JSON original de cada item (mesmo índice que items), para gravação na coluna raw.
type upsertFunc[T any] func(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []T, raws []json.RawMessage) error

// buildPaginacao monta os parâmetros de paginação, aplicando filtro delta se disponível.
// Não aplica filtro quando:
//   - opts.Full = true  → carga completa
//   - opts.IgnorarDelta = true → executor sempre busca tudo (ex: movimentos, contas_correntes)
func buildPaginacao(pagina, pageSize int, opts worker.SyncOptions) omie.PaginacaoParams {
	p := omie.PaginacaoParams{Pagina: pagina, RegistrosPorPagina: pageSize}
	if !opts.Full && !opts.IgnorarDelta && opts.UltimoSyncAt != "" {
		p.FiltrarPorDataDe = opts.UltimoSyncAt
	}
	return p
}

// syncAll pagina a API do Omie e chama upsert a cada página.
func syncAll[T any](
	ctx context.Context,
	log zerolog.Logger,
	client *omie.Client,
	pool *pgxpool.Pool,
	schema string,
	empresaID string,
	jobID string,
	reporter progress.Reporter,
	cfg *omie_config.EndpointConfig,
	fetch pageFunc[T],
	upsert upsertFunc[T],
) error {
	nome := cfg.Modulo
	if !cfg.Ativo {
		log.Warn().Str("modulo", nome).Msg("módulo inativo nas configurações, pulando sync")
		return nil
	}

	if err := reporter.Start(ctx, jobID, nome); err != nil {
		log.Error().Err(err).Str("modulo", nome).Msg("erro ao iniciar reporte de progresso")
	}

	pagina := 1
	total := 0
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	for {
		items, raws, totalPaginas, err := fetch(ctx, client, pagina, pageSize)
		if err != nil {
			// Omie retorna 500 "Não existem registros para a página" quando o filtro
			// incremental não encontra nenhum dado novo. Tratar como sucesso com 0 registros.
			if omie.IsSemRegistros(err) {
				log.Info().Str("modulo", nome).Msg("nenhum registro novo no período (sync incremental)")
				break
			}
			_ = reporter.Fail(ctx, jobID, nome, err, client.LastMaskedPayload, string(client.LastResponseMeta))
			return fmt.Errorf("etl.%s fetch página %d: %w", nome, pagina, err)
		}

		if len(items) > 0 {
			if err := upsert(ctx, pool, schema, empresaID, items, raws); err != nil {
				_ = reporter.Fail(ctx, jobID, nome, err, client.LastMaskedPayload, string(client.LastResponseMeta))
				return fmt.Errorf("etl.%s upsert página %d: %w", nome, pagina, err)
			}
			total += len(items)
		}

		// Atualiza progresso a cada página
		_ = reporter.UpdatePage(ctx, jobID, nome, pagina, totalPaginas, total, totalPaginas*pageSize, client.LastMaskedPayload, client.LastResponseMeta)

		// Heartbeat: registra que o job ainda está vivo
		if err := reporter.Heartbeat(ctx); err != nil {
			log.Warn().Err(err).Msg("falha ao registrar heartbeat")
		}

		log.Debug().
			Str("modulo", nome).
			Int("pagina", pagina).
			Int("total_paginas", totalPaginas).
			Int("registros", len(items)).
			Msg("página sincronizada")

		if pagina >= totalPaginas {
			break
		}
		pagina++
	}

	if err := reporter.Done(ctx, jobID, nome, total); err != nil {
		log.Error().Err(err).Str("modulo", nome).Msg("erro ao finalizar reporte de progresso")
	}

	log.Info().Str("modulo", nome).Int("total", total).Msg("sync concluído")
	return nil
}

// syncPage executa o sync de uma única página.
func syncPage[T any](
	ctx context.Context,
	client *omie.Client,
	pool *pgxpool.Pool,
	schema string,
	empresaID string,
	pagina int,
	cfg *omie_config.EndpointConfig,
	fetch pageFunc[T],
	upsert upsertFunc[T],
) (int, error) {
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	items, raws, _, err := fetch(ctx, client, pagina, pageSize)
	if err != nil {
		if omie.IsSemRegistros(err) {
			return 0, nil
		}
		return 0, err
	}

	if len(items) > 0 {
		if err := upsert(ctx, pool, schema, empresaID, items, raws); err != nil {
			return 0, err
		}
	}

	return len(items), nil
}

// toJSON converte qualquer valor para []byte JSON, usado para a coluna raw.
func toJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// extractArray extrai o array de dados de uma resposta genérica do Omie usando o nome do campo da config.
// Retorna os itens tipados, os itens como JSON bruto original (preserva todos os campos da API) e o total de páginas.
func extractArray[T any](raw map[string]json.RawMessage, fieldName string) ([]T, []json.RawMessage, int, error) {
	// 1. Extrai paginação
	var pag OmieResponse
	if pagRaw, ok := raw["pagina"]; ok {
		_ = json.Unmarshal(pagRaw, &pag.Pagina)
	}
	if tpRaw, ok := raw["total_de_paginas"]; ok {
		_ = json.Unmarshal(tpRaw, &pag.TotalDePaginas)
	}
	if trRaw, ok := raw["total_de_registros"]; ok {
		_ = json.Unmarshal(trRaw, &pag.TotalDeRegistros)
	}

	// 2. Extrai os dados
	dataRaw, ok := raw[fieldName]
	if !ok {
		return nil, nil, 0, fmt.Errorf("campo de dados '%s' não encontrado na resposta", fieldName)
	}

	var items []T
	if err := json.Unmarshal(dataRaw, &items); err != nil {
		return nil, nil, 0, fmt.Errorf("erro ao decodificar array '%s': %w", fieldName, err)
	}

	// Preserva o JSON original de cada elemento para gravação na coluna raw (inclui campos não mapeados no struct).
	var raws []json.RawMessage
	if err := json.Unmarshal(dataRaw, &raws); err != nil {
		return nil, nil, 0, fmt.Errorf("erro ao decodificar raws '%s': %w", fieldName, err)
	}

	tp := pag.TotalDePaginas
	if tp == 0 {
		tp = 1
	}

	return items, raws, tp, nil
}
