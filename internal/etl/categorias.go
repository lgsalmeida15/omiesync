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

type OmieCategoria struct {
	Codigo          string `json:"codigo"`
	Descricao       string `json:"descricao"`
	IdContaCorrente int64  `json:"id_conta_corrente"`
}

type CategoriasExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewCategoriasExecutor(pool *pgxpool.Pool, log zerolog.Logger) *CategoriasExecutor {
	return &CategoriasExecutor{pool: pool, log: log.With().Str("executor", "categorias").Logger()}
}

func (e *CategoriasExecutor) Nome() string { return "categorias" }

func (e *CategoriasExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieCategoria, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieCategoria](resp, cfg.ArrayField)
		},
		upsertCategorias,
	)
}

func (e *CategoriasExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieCategoria, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieCategoria](resp, cfg.ArrayField)
		},
		upsertCategorias,
	)
}

func upsertCategorias(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieCategoria, raws []json.RawMessage) error {
	for i, it := range items {
		if it.Codigo == "" {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.categorias (empresa_id, codigo, descricao, id_conta_corrente, raw, synced_at)
			VALUES ($1,$2,$3,$4,$5,NOW())
			ON CONFLICT (empresa_id, codigo) DO UPDATE SET
				descricao         = EXCLUDED.descricao,
				id_conta_corrente = EXCLUDED.id_conta_corrente,
				raw               = EXCLUDED.raw,
				synced_at         = NOW()
		`, schema),
			empresaID, it.Codigo, it.Descricao, it.IdContaCorrente, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertCategorias [%s]: %w", it.Codigo, err)
		}
	}
	return nil
}
