package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migrationsFS embed.FS

// MigrationResult resume o que foi feito.
type MigrationResult struct {
	Applied []string
	Skipped []string
	Failed  []string
}

// RunMigrations aplica migrations pendentes. Nunca retorna erro fatal —
// falhas individuais são registradas em result.Failed e o servidor continua.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) MigrationResult {
	var result MigrationResult

	conn, err := pool.Acquire(ctx)
	if err != nil {
		slog.Error("db.RunMigrations: falha ao adquirir conexão", "err", err)
		return result
	}
	defer conn.Release()

	// Cria tabela de controle se não existir (requer que _etl já exista)
	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _etl.schema_migrations (
			version     TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		slog.Error("db.RunMigrations: falha ao criar schema_migrations", "err", err)
		return result
	}

	// Baseline: se schema_migrations está vazia mas _etl.grupos existe,
	// o banco foi inicializado manualmente — registra tudo sem re-executar.
	var count int
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM _etl.schema_migrations`).Scan(&count); err != nil {
		slog.Error("db.RunMigrations: falha ao contar versões", "err", err)
		return result
	}
	if count == 0 {
		var gruposExists bool
		_ = conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = '_etl' AND table_name = 'grupos'
			)`).Scan(&gruposExists)

		if gruposExists {
			if entries, err := fs.ReadDir(migrationsFS, "migrations"); err == nil {
				tx, err := conn.Begin(ctx)
				if err == nil {
					for _, e := range entries {
						if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
							v := strings.TrimSuffix(e.Name(), ".up.sql")
							_, _ = tx.Exec(ctx,
								`INSERT INTO _etl.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, v)
							result.Skipped = append(result.Skipped, v)
						}
					}
					if err := tx.Commit(ctx); err != nil {
						_ = tx.Rollback(ctx)
						slog.Error("db.RunMigrations: falha no baseline commit", "err", err)
					}
				}
			}
			return result
		}
	}

	// Lê versões já aplicadas
	rows, err := conn.Query(ctx, `SELECT version FROM _etl.schema_migrations`)
	if err != nil {
		slog.Error("db.RunMigrations: falha ao ler versões", "err", err)
		return result
	}
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err == nil {
			applied[v] = true
		}
	}
	rows.Close()

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		slog.Error("db.RunMigrations: falha ao listar arquivos", "err", err)
		return result
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")
		if applied[version] {
			result.Skipped = append(result.Skipped, version)
			continue
		}

		sql, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			slog.Error("db.RunMigrations: falha ao ler arquivo", "file", name, "err", err)
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", version, err))
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			slog.Error("db.RunMigrations: falha ao iniciar transação", "file", name, "err", err)
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", version, err))
			continue
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			slog.Error("db.RunMigrations: falha ao executar migration", "file", name, "err", err)
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", version, err))
			continue
		}

		if _, err := tx.Exec(ctx, `INSERT INTO _etl.schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			slog.Error("db.RunMigrations: falha ao registrar migration", "file", name, "err", err)
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", version, err))
			continue
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("db.RunMigrations: falha no commit", "file", name, "err", err)
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", version, err))
			continue
		}

		slog.Info("db.RunMigrations: migration aplicada", "version", version)
		result.Applied = append(result.Applied, version)
	}

	return result
}
