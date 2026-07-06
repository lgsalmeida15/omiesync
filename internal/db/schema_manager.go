package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultSchema = "_etl"

// WithSchema retorna um *pgx.Conn com search_path setado para o schema do tenant.
// Sempre fechar a conexão após uso: defer conn.Release() se vier de pool.Acquire.
func WithSchema(ctx context.Context, pool *pgxpool.Pool, schema string) (*pgxpool.Conn, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.WithSchema acquire: %w", err)
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", pgx.Identifier{schema}.Sanitize()))
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("db.WithSchema set search_path: %w", err)
	}

	return conn, nil
}

// SchemaName retorna o nome do schema de um grupo a partir do slug.
func SchemaName(slug string) string {
	return "grupo_" + slug
}
