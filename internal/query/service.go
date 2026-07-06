package query

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/apperror"
)

// Service define as operações do SQL Explorer.
type Service interface {
	ValidateSQL(sql string) error
	Execute(ctx context.Context, pool *pgxpool.Pool, schema, sql string) (*QueryResponse, error)
}

type service struct{}

// NewService cria um novo Service do SQL Explorer.
func NewService() Service {
	return &service{}
}

var forbiddenPrefixes = []string{
	"insert", "update", "delete", "drop", "truncate",
	"alter", "create", "grant", "revoke", "execute",
	"call", "do", "copy", "with",
}

// dangerousFunctions são funções que permitem acesso ao sistema de arquivos ou execução remota.
var dangerousFunctions = []string{
	"pg_read_file", "pg_ls_dir", "pg_execute_server_program", "pg_write_file",
}

// etlPattern detecta referências ao schema _etl mesmo com espaços ou aspas.
var etlPattern = regexp.MustCompile(`(?i)_etl\s*\.|\x22_etl\x22`)

var validSchemaRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var limitRegex = regexp.MustCompile(`(?i)\blimit\s+\d+`)

// ValidateSQL verifica se a query é permitida: apenas SELECT, sem acesso a _etl,
// sem múltiplos statements, sem funções perigosas.
func (s *service) ValidateSQL(sql string) error {
	trimmed := strings.TrimSpace(sql)
	lower := strings.ToLower(trimmed)

	// C2.1 — Rejeitar múltiplos statements via ponto-e-vírgula.
	// Remove o ponto-e-vírgula final (opcional) e verifica se ainda há algum.
	withoutTrailingSemicolon := strings.TrimRight(trimmed, "; \t\r\n")
	if strings.Contains(withoutTrailingSemicolon, ";") {
		return apperror.Unprocessable("múltiplos statements não são permitidos")
	}

	// C2.4 — Verificar prefixo: deve ser SELECT.
	if !strings.HasPrefix(lower, "select") {
		return apperror.Unprocessable("apenas SELECT é permitido")
	}

	for _, prefix := range forbiddenPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return apperror.Unprocessable("apenas SELECT é permitido")
		}
	}

	// C2.2 — Bloquear funções perigosas em qualquer posição do SQL.
	for _, fn := range dangerousFunctions {
		if strings.Contains(lower, fn) {
			return apperror.Forbidden("função não permitida: " + fn)
		}
	}

	// C2.3 — Bloquear referências ao schema _etl com espaços ou aspas.
	if etlPattern.MatchString(trimmed) {
		return apperror.Forbidden("acesso ao schema _etl não é permitido")
	}

	return nil
}

// Execute executa a query validada no schema do grupo, em transação somente leitura.
func (s *service) Execute(ctx context.Context, pool *pgxpool.Pool, schema, sql string) (*QueryResponse, error) {
	// Validar schema contra injeção antes de interpolá-lo.
	if !validSchemaRegex.MatchString(schema) {
		return nil, fmt.Errorf("query.service.Execute: schema inválido: %q", schema)
	}

	// Adiciona LIMIT 1000 se não houver LIMIT explícito, ou usa o existente (máx 1000).
	finalSQL := injectLimit(sql)

	// C1 — Abrir transação já em modo READ ONLY via TxOptions, sem BEGIN avulso.
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("query.service.Execute: begin tx read only: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '30s'"); err != nil {
		return nil, fmt.Errorf("query.service.Execute: set statement_timeout: %w", err)
	}

	// schema já foi validado por validSchemaRegex — interpolação segura.
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path = %s", schema)); err != nil {
		return nil, fmt.Errorf("query.service.Execute: set search_path: %w", err)
	}

	pgxRows, err := tx.Query(ctx, finalSQL)
	if err != nil {
		return nil, fmt.Errorf("query.service.Execute: executar query: %w", err)
	}
	defer pgxRows.Close()

	// Montar colunas a partir das field descriptions.
	fieldDescs := pgxRows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	var allRows [][]any
	for pgxRows.Next() {
		vals, err := pgxRows.Values()
		if err != nil {
			return nil, fmt.Errorf("query.service.Execute: scan row: %w", err)
		}
		allRows = append(allRows, vals)
	}
	if err := pgxRows.Err(); err != nil {
		return nil, fmt.Errorf("query.service.Execute: iteração rows: %w", err)
	}

	rowCount := len(allRows)
	truncated := rowCount == 1000

	if allRows == nil {
		allRows = [][]any{}
	}

	return &QueryResponse{
		Columns:   columns,
		Rows:      allRows,
		RowCount:  rowCount,
		Truncated: truncated,
	}, nil
}

// injectLimit garante que a query tenha no máximo LIMIT 1000.
func injectLimit(sql string) string {
	if limitRegex.MatchString(sql) {
		// Envolve como subquery para sobrescrever qualquer LIMIT existente.
		return fmt.Sprintf("SELECT * FROM (%s) AS _q LIMIT 1000", sql)
	}
	return fmt.Sprintf("%s LIMIT 1000", sql)
}
