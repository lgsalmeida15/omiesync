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

type OmieOrdemServico struct {
	NumeroOS       int64   `json:"nNumOS"`
	DataAbertura   string  `json:"dDtAbertura"`
	DataPrevisao   string  `json:"dDtPrevisao"`
	DataFechamento string  `json:"dDtFechamento"`
	Status         string  `json:"cStatus"`
	CodigoCliente  int64   `json:"nCodCliente"`
	ValorTotal     float64 `json:"nValorTotal"`
	Descricao      string  `json:"cDescricao"`
}

type OrdensServicoExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewOrdensServicoExecutor(pool *pgxpool.Pool, log zerolog.Logger) *OrdensServicoExecutor {
	return &OrdensServicoExecutor{pool: pool, log: log.With().Str("executor", "ordens_servico").Logger()}
}

func (e *OrdensServicoExecutor) Nome() string { return "ordens_servico" }

func (e *OrdensServicoExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieOrdemServico, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieOrdemServico](resp, cfg.ArrayField)
		},
		upsertOrdensServico,
	)
}

func (e *OrdensServicoExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieOrdemServico, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieOrdemServico](resp, cfg.ArrayField)
		},
		upsertOrdensServico,
	)
}

func upsertOrdensServico(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieOrdemServico, raws []json.RawMessage) error {
	for i, it := range items {
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.ordens_servico
				(empresa_id, numero_os, data_abertura, data_previsao, data_fechamento,
				 status, codigo_cliente, valor_total, descricao, raw, synced_at)
			VALUES ($1,$2,
				NULLIF($3,'')::date, NULLIF($4,'')::date, NULLIF($5,'')::date,
				$6,$7,$8,$9,$10,NOW())
			ON CONFLICT (empresa_id, numero_os) DO UPDATE SET
				data_abertura   = EXCLUDED.data_abertura,
				data_previsao   = EXCLUDED.data_previsao,
				data_fechamento = EXCLUDED.data_fechamento,
				status          = EXCLUDED.status,
				codigo_cliente  = EXCLUDED.codigo_cliente,
				valor_total     = EXCLUDED.valor_total,
				descricao       = EXCLUDED.descricao,
				raw             = EXCLUDED.raw,
				synced_at       = NOW()
		`, schema),
			empresaID, it.NumeroOS, parseOmieDate(it.DataAbertura), parseOmieDate(it.DataPrevisao), parseOmieDate(it.DataFechamento),
			it.Status, it.CodigoCliente, it.ValorTotal, it.Descricao, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertOrdensServico [%d]: %w", it.NumeroOS, err)
		}
	}
	return nil
}


