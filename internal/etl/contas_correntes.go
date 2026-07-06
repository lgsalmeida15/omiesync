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

type OmieContaCorrente struct {
	CodigoContaCorrente int64   `json:"nCodCC"`
	Descricao           string  `json:"descricao"`
	Tipo                string  `json:"tipo"`
	SaldoInicial        float64 `json:"saldo_inicial"`
}

type ContasCorrentesExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewContasCorrentesExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ContasCorrentesExecutor {
	return &ContasCorrentesExecutor{pool: pool, log: log.With().Str("executor", "contas_correntes").Logger()}
}

func (e *ContasCorrentesExecutor) Nome() string { return "contas_correntes" }

func (e *ContasCorrentesExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaCorrente, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaCorrente](resp, cfg.ArrayField)
		},
		upsertContasCorrentes,
	)
}

func (e *ContasCorrentesExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContaCorrente, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContaCorrente](resp, cfg.ArrayField)
		},
		upsertContasCorrentes,
	)
}

func upsertContasCorrentes(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieContaCorrente, raws []json.RawMessage) error {
	for i, it := range items {
		if it.CodigoContaCorrente == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.contas_correntes
				(empresa_id, codigo_conta_corrente, descricao, tipo, saldo_inicial, raw, synced_at)
			VALUES ($1,$2,$3,$4,$5,$6,NOW())
			ON CONFLICT (empresa_id, codigo_conta_corrente) DO UPDATE SET
				descricao     = EXCLUDED.descricao,
				tipo          = EXCLUDED.tipo,
				saldo_inicial = EXCLUDED.saldo_inicial,
				raw           = EXCLUDED.raw,
				synced_at     = NOW()
		`, schema),
			empresaID, it.CodigoContaCorrente, it.Descricao, it.Tipo, it.SaldoInicial, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertContasCorrentes [%d]: %w", it.CodigoContaCorrente, err)
		}
	}
	return nil
}
