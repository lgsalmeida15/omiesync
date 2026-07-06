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

type OmieDepartamento struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
	Estrutura string `json:"estrutura"`
	Inativo   string `json:"inativo"`
}

type DepartamentosExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewDepartamentosExecutor(pool *pgxpool.Pool, log zerolog.Logger) *DepartamentosExecutor {
	return &DepartamentosExecutor{pool: pool, log: log.With().Str("executor", "departamentos").Logger()}
}

func (e *DepartamentosExecutor) Nome() string { return "departamentos" }

func (e *DepartamentosExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieDepartamento, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieDepartamento](resp, cfg.ArrayField)
		},
		upsertDepartamentos,
	)
}

func (e *DepartamentosExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieDepartamento, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieDepartamento](resp, cfg.ArrayField)
		},
		upsertDepartamentos,
	)
}

func upsertDepartamentos(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieDepartamento, raws []json.RawMessage) error {
	for i, it := range items {
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.departamentos (empresa_id, codigo, descricao, inativo, raw, synced_at)
			VALUES ($1,$2,$3,$4,$5,NOW())
			ON CONFLICT (empresa_id, codigo) DO UPDATE SET
				descricao = EXCLUDED.descricao,
				inativo   = EXCLUDED.inativo,
				raw       = EXCLUDED.raw,
				synced_at = NOW()
		`, schema),
			empresaID, it.Codigo, it.Descricao, it.Inativo, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertDepartamentos [%s]: %w", it.Codigo, err)
		}
	}
	return nil
}


