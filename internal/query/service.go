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

// writeKeywords são comandos de escrita. Verificados em QUALQUER posição, não só no
// prefixo, para barrar CTE que modifica dados — WITH x AS (DELETE ...) SELECT ... — que
// de outro modo passaria pelo teste de prefixo.
//
// O casamento é por palavra inteira: "update" não casa com a coluna "updated_at",
// nem "delete" com "deleted_at".
var writeKeywordRegex = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|truncate` +
	`|alter|create|grant|revoke|execute|call|copy|merge|refresh|vacuum|reindex)\b`)

// explainPrefixRegex remove o cabeçalho EXPLAIN [(opções)] [ANALYZE] [VERBOSE] para
// que o statement interno seja validado como qualquer outra consulta.
var explainPrefixRegex = regexp.MustCompile(`(?is)^explain\s*(\([^)]*\)\s*)?((analyze|verbose)\s+)*`)

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

	// EXPLAIN é análise, não escrita: remove o cabeçalho e valida o statement interno.
	// Mesmo EXPLAIN ANALYZE, que de fato executa, é inofensivo aqui — a transação é
	// READ ONLY e o statement_timeout de 30s continua valendo.
	corpo := lower
	if explainPrefixRegex.MatchString(corpo) {
		corpo = strings.TrimSpace(explainPrefixRegex.ReplaceAllString(corpo, ""))
		if corpo == "" {
			return apperror.Unprocessable("EXPLAIN exige uma consulta")
		}
	}

	// Aceita SELECT e WITH (CTE). WITH era rejeitado por prefixo, o que barrava
	// consultas de leitura legítimas — a proteção real contra CTE de escrita é a
	// varredura de palavras-chave abaixo, somada à transação READ ONLY.
	if !strings.HasPrefix(corpo, "select") && !strings.HasPrefix(corpo, "with") {
		return apperror.Unprocessable("apenas SELECT, WITH ou EXPLAIN são permitidos")
	}

	// Palavras de escrita em qualquer posição — pega CTE modificadora.
	if m := writeKeywordRegex.FindString(corpo); m != "" {
		return apperror.Unprocessable("comando não permitido: " + strings.ToUpper(m))
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
//
// EXPLAIN fica de fora: o plano tem poucas linhas e nem SELECT ... FROM (EXPLAIN ...)
// nem EXPLAIN ... LIMIT são SQL válido.
func injectLimit(sql string) string {
	if explainPrefixRegex.MatchString(strings.ToLower(strings.TrimSpace(sql))) {
		return sql
	}
	if limitRegex.MatchString(sql) {
		// Envolve como subquery para sobrescrever qualquer LIMIT existente.
		return fmt.Sprintf("SELECT * FROM (%s) AS _q LIMIT 1000", sql)
	}
	return fmt.Sprintf("%s LIMIT 1000", sql)
}
