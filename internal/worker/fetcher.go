package worker

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type empresaFetcher struct {
	pool *pgxpool.Pool
}

func NewEmpresaFetcher(pool *pgxpool.Pool) EmpresaFetcher {
	return &empresaFetcher{pool: pool}
}

func (f *empresaFetcher) PausarEmpresa(ctx context.Context, empresaID string) error {
	_, err := f.pool.Exec(ctx, `
		UPDATE _etl.empresas
		SET status_sync = 'pausado', updated_at = NOW()
		WHERE id = $1
	`, empresaID)
	if err != nil {
		return fmt.Errorf("worker.fetcher.PausarEmpresa: %w", err)
	}
	return nil
}

func (f *empresaFetcher) GetActiveCredentials(ctx context.Context, empresaID string) (*EmpresaCredentials, error) {
	row := f.pool.QueryRow(ctx, `
		SELECT e.id::text, g.id::text, e.app_key, e.app_secret, g.schema_name
		FROM _etl.empresas e
		JOIN _etl.grupos g ON g.id = e.grupo_id
		WHERE e.id = $1
		  AND e.status = 'ativa'
		  AND e.deleted_at IS NULL
	`, empresaID)

	var c EmpresaCredentials
	if err := row.Scan(&c.ID, &c.GrupoID, &c.AppKey, &c.AppSecret, &c.Schema); err != nil {
		return nil, fmt.Errorf("worker.fetcher.GetActiveCredentials: %w", err)
	}
	return &c, nil
}
