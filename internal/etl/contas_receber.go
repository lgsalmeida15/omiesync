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

type OmieContaReceber struct {
	CodigoLancamento  int64   `json:"codigo_lancamento_omie"`
	DataVencimento    string  `json:"data_vencimento"`
	DataPrevisao      string  `json:"data_previsao"`
	DataRecebimento   string  `json:"data_pagamento"`
	ValorDocumento    float64 `json:"valor_documento"`
	ValorRecebido     float64 `json:"valor_pago"`
	StatusTitulo      string  `json:"status_titulo"`
	CodigoCliente     int64   `json:"codigo_cliente_fornecedor"`
	CodigoCategoria   string  `json:"codigo_categoria"`
	Observacao        string  `json:"observacao"`
}

type ContasReceberExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewContasReceberExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ContasReceberExecutor {
	return &ContasReceberExecutor{pool: pool, log: log.With().Str("executor", "contas_receber").Logger()}
}

func (e *ContasReceberExecutor) Nome() string { return "contas_receber" }

func (e *ContasReceberExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaReceber, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaReceber](resp, cfg.ArrayField)
		},
		upsertContasReceber,
	)
}

func (e *ContasReceberExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaReceber, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaReceber](resp, cfg.ArrayField)
		},
		upsertContasReceber,
	)
}

func upsertContasReceber(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieContaReceber, raws []json.RawMessage) error {
	for i, it := range items {
		if it.CodigoLancamento == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.contas_receber
				(empresa_id, codigo_lancamento, data_vencimento, data_previsao, data_recebimento,
				 valor_documento, valor_recebido, status_titulo, codigo_cliente,
				 codigo_categoria, observacao, raw, synced_at)
			VALUES ($1,$2,
				NULLIF($3,'')::date, NULLIF($4,'')::date, NULLIF($5,'')::date,
				$6,$7,$8,$9,$10,$11,$12,NOW())
			ON CONFLICT (empresa_id, codigo_lancamento) DO UPDATE SET
				data_vencimento  = EXCLUDED.data_vencimento,
				data_previsao    = EXCLUDED.data_previsao,
				data_recebimento = EXCLUDED.data_recebimento,
				valor_documento  = EXCLUDED.valor_documento,
				valor_recebido   = EXCLUDED.valor_recebido,
				status_titulo    = EXCLUDED.status_titulo,
				codigo_cliente   = EXCLUDED.codigo_cliente,
				codigo_categoria = EXCLUDED.codigo_categoria,
				observacao       = EXCLUDED.observacao,
				raw              = EXCLUDED.raw,
				synced_at        = NOW()
		`, schema),
			empresaID, it.CodigoLancamento, parseOmieDate(it.DataVencimento), parseOmieDate(it.DataPrevisao), parseOmieDate(it.DataRecebimento),
			it.ValorDocumento, it.ValorRecebido, it.StatusTitulo, it.CodigoCliente,
			it.CodigoCategoria, it.Observacao, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertContasReceber [%d]: %w", it.CodigoLancamento, err)
		}
	}
	return nil
}
