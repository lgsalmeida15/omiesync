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

type OmieProjeto struct {
	CodigoProjeto int64  `json:"codigo"`
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	DataInicio    string `json:"dDtInicio"`
	DataFim       string `json:"dDtFim"`
	Status        string `json:"inativo"`
}

type ProjetosExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewProjetosExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ProjetosExecutor {
	return &ProjetosExecutor{pool: pool, log: log.With().Str("executor", "projetos").Logger()}
}

func (e *ProjetosExecutor) Nome() string { return "projetos" }

func (e *ProjetosExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieProjeto, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieProjeto](resp, cfg.ArrayField)
		},
		upsertProjetos,
	)
}

func (e *ProjetosExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieProjeto, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieProjeto](resp, cfg.ArrayField)
		},
		upsertProjetos,
	)
}

func upsertProjetos(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieProjeto, raws []json.RawMessage) error {
	for i, it := range items {
		if it.CodigoProjeto == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.projetos
				(empresa_id, codigo_projeto, nome, descricao, data_inicio, data_fim, status, raw, synced_at)
			VALUES ($1,$2,$3,$4,
				NULLIF($5,'')::date, NULLIF($6,'')::date,
				$7,$8,NOW())
			ON CONFLICT (empresa_id, codigo_projeto) DO UPDATE SET
				nome        = EXCLUDED.nome,
				descricao   = EXCLUDED.descricao,
				data_inicio = EXCLUDED.data_inicio,
				data_fim    = EXCLUDED.data_fim,
				status      = EXCLUDED.status,
				raw         = EXCLUDED.raw,
				synced_at   = NOW()
		`, schema),
			empresaID, it.CodigoProjeto, it.Nome, it.Descricao,
			parseOmieDate(it.DataInicio), parseOmieDate(it.DataFim), it.Status, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertProjetos [%d]: %w", it.CodigoProjeto, err)
		}
	}
	return nil
}
