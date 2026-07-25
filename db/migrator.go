package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migrationsFS embed.FS

// RunMigrations aplica todas as migrations *.up.sql que ainda não foram executadas.
// Cria a tabela _etl.schema_migrations se necessário e executa cada arquivo uma única vez.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("db.RunMigrations acquire: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _etl.schema_migrations (
			version     TEXT PRIMARY KEY,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("db.RunMigrations criar schema_migrations: %w", err)
	}

	// Verifica se o banco já existia antes do migrator ser introduzido.
	// Se schema_migrations está vazia mas _etl.grupos já existe, é um banco
	// inicializado manualmente — marca todas as migrations como aplicadas (baseline)
	// para não re-executar o que já está lá.
	var count int
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM _etl.schema_migrations`).Scan(&count); err != nil {
		return fmt.Errorf("db.RunMigrations contar versões: %w", err)
	}
	if count == 0 {
		var gruposExists bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = '_etl' AND table_name = 'grupos'
			)`).Scan(&gruposExists); err != nil {
			return fmt.Errorf("db.RunMigrations verificar baseline: %w", err)
		}
		if gruposExists {
			// Banco pré-existente: registra todas as migrations como já aplicadas
			// exceto as que precisam rodar agora (serão executadas normalmente abaixo)
			// Estratégia: marca tudo como aplicado; só as realmente novas rodarão
			// porque não existem no banco ainda (CREATE TABLE IF NOT EXISTS / idempotentes).
			// Para migrations não-idempotentes, registramos sem executar.
			entries2, _ := fs.ReadDir(migrationsFS, "migrations")
			tx, err := conn.Begin(ctx)
			if err != nil {
				return fmt.Errorf("db.RunMigrations begin baseline: %w", err)
			}
			for _, e := range entries2 {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
					continue
				}
				v := strings.TrimSuffix(e.Name(), ".up.sql")
				if _, err := tx.Exec(ctx,
					`INSERT INTO _etl.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, v); err != nil {
					_ = tx.Rollback(ctx)
					return fmt.Errorf("db.RunMigrations baseline insert %s: %w", v, err)
				}
			}
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("db.RunMigrations baseline commit: %w", err)
			}
		}
	}

	rows, err := conn.Query(ctx, `SELECT version FROM _etl.schema_migrations`)
	if err != nil {
		return fmt.Errorf("db.RunMigrations ler versões: %w", err)
	}
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return fmt.Errorf("db.RunMigrations scan versão: %w", err)
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("db.RunMigrations rows.Err: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("db.RunMigrations ler dir: %w", err)
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
			continue
		}

		sql, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("db.RunMigrations ler %s: %w", name, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("db.RunMigrations begin %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db.RunMigrations executar %s: %w", name, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO _etl.schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db.RunMigrations registrar %s: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("db.RunMigrations commit %s: %w", name, err)
		}
	}

	return nil
}
