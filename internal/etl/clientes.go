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

// --- Tipos Omie ---

type OmieCliente struct {
	CodigoClienteOmie int64  `json:"codigo_cliente_omie"`
	RazaoSocial       string `json:"razao_social"`
	NomeFantasia      string `json:"nome_fantasia"`
	CnpjCpf           string `json:"cnpj_cpf"`
	Email             string `json:"email"`
	Telefone1DDD      string `json:"telefone1_ddd"`
	Telefone1Numero   string `json:"telefone1_numero"`
	Endereco          string `json:"endereco"`
	Cidade            string `json:"cidade"`
	Estado            string `json:"estado"`
	Cep               string `json:"cep"`
	Inativo           string `json:"inativo"`
	DataAlteracao     string `json:"data_alteracao"`
}

// --- Executor ---

type ClientesExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewClientesExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ClientesExecutor {
	return &ClientesExecutor{pool: pool, log: log.With().Str("executor", "clientes").Logger()}
}

func (e *ClientesExecutor) Nome() string { return "clientes" }

func (e *ClientesExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieCliente, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieCliente](resp, cfg.ArrayField)
		},
		upsertClientes,
	)
}

func (e *ClientesExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieCliente, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildPaginacao(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieCliente](resp, cfg.ArrayField)
		},
		upsertClientes,
	)
}

func upsertClientes(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieCliente, raws []json.RawMessage) error {
	for i, it := range items {
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.clientes
				(empresa_id, codigo_cliente_omie, razao_social, nome_fantasia, cnpj_cpf,
				 email, telefone1_ddd, telefone1_numero, endereco, cidade,
				 estado, cep, ativo, data_alteracao, raw, synced_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
			ON CONFLICT (empresa_id, codigo_cliente_omie) DO UPDATE SET
				razao_social     = EXCLUDED.razao_social,
				nome_fantasia    = EXCLUDED.nome_fantasia,
				cnpj_cpf         = EXCLUDED.cnpj_cpf,
				email            = EXCLUDED.email,
				telefone1_ddd    = EXCLUDED.telefone1_ddd,
				telefone1_numero = EXCLUDED.telefone1_numero,
				endereco         = EXCLUDED.endereco,
				cidade           = EXCLUDED.cidade,
				estado           = EXCLUDED.estado,
				cep              = EXCLUDED.cep,
				ativo            = EXCLUDED.ativo,
				data_alteracao   = EXCLUDED.data_alteracao,
				raw              = EXCLUDED.raw,
				synced_at        = NOW()
		`, schema),
			empresaID, it.CodigoClienteOmie, it.RazaoSocial, it.NomeFantasia, it.CnpjCpf,
			it.Email, it.Telefone1DDD, it.Telefone1Numero, it.Endereco, it.Cidade,
			it.Estado, it.Cep, it.Inativo != "S", it.DataAlteracao, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertClientes [%d]: %w", it.CodigoClienteOmie, err)
		}
	}
	return nil
}


