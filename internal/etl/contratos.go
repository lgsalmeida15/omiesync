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

// OmieContratoCabecalho são os dados de identificação do contrato de serviço.
// Diferente dos demais módulos, o ListarContratos aninha os campos principais
// dentro de "cabecalho" em vez de deixá-los na raiz do elemento.
type OmieContratoCabecalho struct {
	CodigoContrato  int64   `json:"nCodCtr"`
	CodigoCliente   int64   `json:"nCodCli"`
	NumeroContrato  string  `json:"cNumCtr"`
	CodigoSituacao  string  `json:"cCodSit"`
	VigenciaInicial string  `json:"dVigInicial"`
	VigenciaFinal   string  `json:"dVigFinal"`
	DiaFaturamento  int     `json:"nDiaFat"`
	ValorTotalMes   float64 `json:"nValTotMes"`
	TipoFaturamento string  `json:"cTipoFat"`
}

// OmieContratoInfAdic traz categoria, projeto e conta corrente padrão do contrato.
type OmieContratoInfAdic struct {
	CodigoCategoria     string `json:"cCodCateg"`
	CodigoProjeto       int64  `json:"nCodProj"`
	CodigoContaCorrente int64  `json:"nCodCC"`
}

// OmieContrato mapeia só o que vira coluna tipada. O rateio por departamento
// (departamentos[]) e as categorias por item (itensContrato[].itemCabecalho.cCodCategItem)
// ficam em raw, no mesmo padrão de contas_pagar.raw -> 'categorias' / 'distribuicao'.
type OmieContrato struct {
	Cabecalho OmieContratoCabecalho `json:"cabecalho"`
	InfAdic   OmieContratoInfAdic   `json:"infAdic"`
}

type ContratosExecutor struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewContratosExecutor(pool *pgxpool.Pool, log zerolog.Logger) *ContratosExecutor {
	return &ContratosExecutor{pool: pool, log: log.With().Str("executor", "contratos").Logger()}
}

func (e *ContratosExecutor) Nome() string { return "contratos" }

// buildContratosParams acrescenta apenas_importado_api à paginação padrão.
// O ListarContratos exige o campo; "N" traz todos os contratos, não só os
// criados pela API. O filtro delta de buildPaginacao continua valendo.
func buildContratosParams(pagina, pageSize int, opts worker.SyncOptions) omie.PaginacaoParams {
	p := buildPaginacao(pagina, pageSize, opts)
	p.ApenasImportadoAPI = "N"
	return p
}

func (e *ContratosExecutor) Execute(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, jobID string, rep progress.Reporter, cfg *omie_config.EndpointConfig) error {
	return syncAll(ctx, e.log, client, e.pool, schema, opts.EmpresaID, jobID, rep, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContrato, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildContratosParams(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContrato](resp, cfg.ArrayField)
		},
		upsertContratos,
	)
}

func (e *ContratosExecutor) ExecutePage(ctx context.Context, client *omie.Client, schema string, opts worker.SyncOptions, pagina int, cfg *omie_config.EndpointConfig) (int, error) {
	return syncPage(ctx, client, e.pool, schema, opts.EmpresaID, pagina, cfg,
		func(ctx context.Context, c *omie.Client, pagina, pageSize int) ([]OmieContrato, []json.RawMessage, int, error) {
			var resp map[string]json.RawMessage
			err := c.CallPublic(ctx, cfg.EndpointPath, cfg.Action, buildContratosParams(pagina, pageSize, opts), &resp)
			if err != nil {
				return nil, nil, 0, err
			}
			return extractArray[OmieContrato](resp, cfg.ArrayField)
		},
		upsertContratos,
	)
}

func upsertContratos(ctx context.Context, pool *pgxpool.Pool, schema string, empresaID string, items []OmieContrato, raws []json.RawMessage) error {
	for i, it := range items {
		cab := it.Cabecalho
		if cab.CodigoContrato == 0 {
			continue
		}
		raw := toJSON(it)
		if i < len(raws) {
			raw = raws[i]
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.contratos
				(empresa_id, codigo_lancamento_omie, numero_contrato, codigo_cliente,
				 codigo_situacao, vigencia_inicial, vigencia_final, dia_faturamento,
				 valor_total_mes, tipo_faturamento, codigo_categoria, codigo_projeto,
				 codigo_conta_corrente, raw, synced_at)
			VALUES ($1,$2,$3,$4,$5,
				NULLIF($6,'')::date, NULLIF($7,'')::date,
				$8,$9,$10,$11,$12,$13,$14,NOW())
			ON CONFLICT (empresa_id, codigo_lancamento_omie) DO UPDATE SET
				numero_contrato       = EXCLUDED.numero_contrato,
				codigo_cliente        = EXCLUDED.codigo_cliente,
				codigo_situacao       = EXCLUDED.codigo_situacao,
				vigencia_inicial      = EXCLUDED.vigencia_inicial,
				vigencia_final        = EXCLUDED.vigencia_final,
				dia_faturamento       = EXCLUDED.dia_faturamento,
				valor_total_mes       = EXCLUDED.valor_total_mes,
				tipo_faturamento      = EXCLUDED.tipo_faturamento,
				codigo_categoria      = EXCLUDED.codigo_categoria,
				codigo_projeto        = EXCLUDED.codigo_projeto,
				codigo_conta_corrente = EXCLUDED.codigo_conta_corrente,
				raw                   = EXCLUDED.raw,
				synced_at             = NOW()
		`, schema),
			empresaID, cab.CodigoContrato, cab.NumeroContrato, cab.CodigoCliente,
			cab.CodigoSituacao, parseOmieDate(cab.VigenciaInicial), parseOmieDate(cab.VigenciaFinal),
			cab.DiaFaturamento, cab.ValorTotalMes, cab.TipoFaturamento,
			it.InfAdic.CodigoCategoria, it.InfAdic.CodigoProjeto, it.InfAdic.CodigoContaCorrente, raw,
		)
		if err != nil {
			return fmt.Errorf("upsertContratos [%d]: %w", cab.CodigoContrato, err)
		}
	}
	return nil
}
