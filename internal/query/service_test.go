package query

import "testing"

func TestValidateSQL(t *testing.T) {
	s := NewService()

	casos := []struct {
		nome   string
		sql    string
		aceita bool
	}{
		// ── Permitidos ──────────────────────────────────────────────────────
		{"select simples", "SELECT 1", true},
		{"select de tabela", "SELECT * FROM clientes", true},
		{"cte de leitura", "WITH x AS (SELECT 1 AS a) SELECT * FROM x", true},
		{"cte encadeada", "WITH a AS (SELECT 1), b AS (SELECT 2) SELECT * FROM a, b", true},
		{"explain", "EXPLAIN SELECT * FROM clientes", true},
		{"explain analyze", "EXPLAIN ANALYZE SELECT * FROM clientes", true},
		{"explain com opcoes", "EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM clientes", true},
		{"explain de cte", "EXPLAIN WITH x AS (SELECT 1) SELECT * FROM x", true},
		{"ponto e virgula final", "SELECT 1;", true},

		// Colunas cujo nome contém palavra-chave não podem ser confundidas com
		// comando de escrita — é o falso-positivo clássico da varredura por texto.
		{"coluna updated_at", "SELECT updated_at FROM clientes", true},
		{"coluna deleted_at", "SELECT deleted_at FROM empresas", true},
		{"coluna created_at", "SELECT created_at, updated_at FROM clientes", true},

		// ── Bloqueados ──────────────────────────────────────────────────────
		{"insert", "INSERT INTO clientes VALUES (1)", false},
		{"update", "UPDATE clientes SET nome = 'x'", false},
		{"delete", "DELETE FROM clientes", false},
		{"drop", "DROP TABLE clientes", false},
		{"truncate", "TRUNCATE clientes", false},
		{"alter", "ALTER TABLE clientes ADD COLUMN x INT", false},
		{"refresh matvw", "REFRESH MATERIALIZED VIEW matvw_gerencial_ano_corrente", false},

		// CTE que modifica dados: passava no teste de prefixo antigo, porque começa
		// com WITH. É a razão de a varredura ser em qualquer posição.
		{"cte com delete", "WITH x AS (DELETE FROM clientes RETURNING *) SELECT * FROM x", false},
		{"cte com insert", "WITH x AS (INSERT INTO clientes VALUES (1) RETURNING *) SELECT * FROM x", false},
		{"cte com update", "WITH x AS (UPDATE clientes SET nome='y' RETURNING *) SELECT * FROM x", false},
		{"explain escondendo delete", "EXPLAIN DELETE FROM clientes", false},

		{"multiplos statements", "SELECT 1; DROP TABLE clientes", false},
		{"acesso a _etl", "SELECT * FROM _etl.empresas", false},
		{"acesso a _etl com espaco", "SELECT * FROM _etl . usuarios", false},
		{"acesso a _etl com aspas", `SELECT * FROM "_etl".usuarios`, false},
		{"funcao perigosa", "SELECT pg_read_file('/etc/passwd')", false},
		{"explain sem consulta", "EXPLAIN", false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := s.ValidateSQL(c.sql)
			if c.aceita && err != nil {
				t.Errorf("deveria aceitar, mas rejeitou: %v\nSQL: %s", err, c.sql)
			}
			if !c.aceita && err == nil {
				t.Errorf("deveria rejeitar, mas aceitou\nSQL: %s", c.sql)
			}
		})
	}
}

func TestInjectLimit(t *testing.T) {
	casos := []struct {
		nome     string
		sql      string
		esperado string
	}{
		{"sem limit", "SELECT 1", "SELECT 1 LIMIT 1000"},
		{"com limit", "SELECT 1 LIMIT 5", "SELECT * FROM (SELECT 1 LIMIT 5) AS _q LIMIT 1000"},
		// EXPLAIN não aceita LIMIT nem pode virar subquery.
		{"explain intacto", "EXPLAIN SELECT 1", "EXPLAIN SELECT 1"},
		{"explain analyze intacto", "EXPLAIN ANALYZE SELECT 1", "EXPLAIN ANALYZE SELECT 1"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := injectLimit(c.sql); got != c.esperado {
				t.Errorf("got  %q\nwant %q", got, c.esperado)
			}
		})
	}
}
