package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return NewPoolWithConcurrency(ctx, databaseURL, 20)
}

// NewPoolWithConcurrency cria o pool dimensionado para o número de workers simultâneos.
// MaxConns = maxConcurrent × 3 + 10 (3 conexões por worker + folga para handlers HTTP).
func NewPoolWithConcurrency(ctx context.Context, databaseURL string, maxConcurrent int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool parse config: %w", err)
	}

	cfg.MaxConns = int32(maxConcurrent*3 + 10)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db.NewPool ping: %w", err)
	}

	return pool, nil
}
