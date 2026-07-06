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

type OmieContaPagar struct {
	CodigoLancamento    int64   `json:"codigo_lancamento_omie"`
	DataVencimento      string  `json:"data_vencimento"`
	DataPrevisao        string  `json:"data_previsao"`
	DataPagamento       string  `json:"data_pagamento"`
	ValorDocumento      float64 `json:"valor_documento"`
	ValorPago           float64 `json:"valor_pago"`
	StatusTitulo        string  `json:"status_titulo"`
	CodigoCliente       int64   `json:"codigo_cliente_fornecedor"`
	CodigoCategoria     string  `json:"codigo_categoria"`
	Observacao          string  `json:"observacao"`
}

type ContasPagarExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewContasPagarExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ContasPagarExecutor {
	return &ContasPagarExecutor{pool: pool, log: log.With().Str("executor", "contas_pagar").Logger()}
}

func (e *ContasPagarExecutor) Nome() string { return "contas_pagar" }

func (e *ContasPagarExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaPagar, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaPagar](resp, cfg.ArrayField)
		},
		upsertContasPagar,
	)
}

func (e *ContasPagarExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaPagar, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaPagar](resp, cfg.ArrayField)
		},
		upsertContasPagar,
	)
}

func upsertContasPagar(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieContaPagar, raws []json.RawMessage) error {
	for i, it := range items {
		if it.CodigoLancamento == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.contas_pagar
				(empresa_id, codigo_lancamento, data_vencimento, data_previsao, data_pagamento,
				 valor_documento, valor_pago, status_titulo, codigo_cliente,
				 codigo_categoria, observacao, raw, synced_at)
			VALUES ($1,$2,
				NULLIF($3,'')::DATE, NULLIF($4,'')::DATE, NULLIF($5,'')::DATE,
				$6,$7,$8,$9,$10,$11,$12,NOW())
			ON CONFLICT (empresa_id, codigo_lancamento) DO UPDATE SET
				data_vencimento  = EXCLUDED.data_vencimento,
				data_previsao    = EXCLUDED.data_previsao,
				data_pagamento   = EXCLUDED.data_pagamento,
				valor_documento  = EXCLUDED.valor_documento,
				valor_pago       = EXCLUDED.valor_pago,
				status_titulo    = EXCLUDED.status_titulo,
				codigo_cliente   = EXCLUDED.codigo_cliente,
				codigo_categoria = EXCLUDED.codigo_categoria,
				observacao       = EXCLUDED.observacao,
				raw              = EXCLUDED.raw,
				synced_at        = NOW()
		`, schema),
			empresaID, it.CodigoLancamento, it.DataVencimento, it.DataPrevisao, it.DataPagamento,
			it.ValorDocumento, it.ValorPago, it.StatusTitulo, it.CodigoCliente,
			it.CodigoCategoria, it.Observacao, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertContasPagar [%d]: %w", it.CodigoLancamento, err)
		}
	}
	return nil
}
