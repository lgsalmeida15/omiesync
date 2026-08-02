package etl

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Teste de integração do replaceMovimentos. Requer um Postgres real:
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/etl/ -run Integration -v
//
// Sem a variável, é pulado — não quebra CI nem `go test ./...` local.
//
// Cobre o risco central da carga completa: a tabela não tem chave de negócio, então
// a corretude depende inteiramente da transação DELETE + COPY. Um sync repetido
// duplicaria tudo se ela falhasse.
func TestIntegration_ReplaceMovimentos(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL não definida — teste de integração pulado")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexão: %v", err)
	}
	defer pool.Close()

	const schema = "etl_it"
	const empresaA = "44444444-4444-4444-4444-444444444444"
	const empresaB = "55555555-5555-5555-5555-555555555555"

	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE; CREATE SCHEMA `+schema+`;
		CREATE TABLE `+schema+`.movimentos_financeiros (
			id BIGSERIAL PRIMARY KEY, empresa_id UUID NOT NULL,
			codigo_titulo BIGINT, numero_titulo TEXT, codigo_mov_cc BIGINT,
			codigo_mov_cc_repet BIGINT, codigo_titrepet BIGINT,
			data_lancamento DATE, valor NUMERIC(15,2), tipo TEXT,
			codigo_conta_corrente BIGINT, codigo_categoria TEXT, raw JSONB,
			synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer func() { _, _ = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`) }()

	mov := func(titulo, movCC int64, valorTitulo, valorMovCC float64) OmieMovimento {
		return OmieMovimento{Detalhes: OmieMovimentoDetalhes{
			CodigoTitulo: titulo, CodigoMovCC: movCC,
			ValorTitulo: valorTitulo, ValorMovCC: valorMovCC,
			DataRegistro: "02/01/2026", Grupo: "CONTA_CORRENTE_PAG", Natureza: "P",
		}}
	}
	// Sem dDtRegistro: o COPY é binário e não aceita "" numa coluna DATE — precisa
	// de NULL de verdade. Foi o que quebrou o primeiro sync full em produção.
	movSemData := func(titulo, movCC int64) OmieMovimento {
		m := mov(titulo, movCC, 1, 0)
		m.Detalhes.DataRegistro = ""
		return m
	}
	raws := func(n int) []json.RawMessage {
		out := make([]json.RawMessage, n)
		for i := range out {
			out[i] = json.RawMessage(`{"detalhes":{}}`)
		}
		return out
	}

	conta := func(empresa string) int {
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM `+schema+`.movimentos_financeiros WHERE empresa_id = $1`,
			empresa).Scan(&n); err != nil {
			t.Fatalf("contagem: %v", err)
		}
		return n
	}

	itens := []OmieMovimento{
		mov(1, 100, 10, 0),
		mov(2, 100, 20, 0),
		mov(0, 300, 0, 5.5),
		movSemData(3, 400),
	}

	// 1º sync
	n, err := replaceMovimentos(ctx, pool, schema, empresaA, itens, raws(len(itens)))
	if err != nil {
		t.Fatalf("1º sync: %v", err)
	}
	if n != 4 || conta(empresaA) != 4 {
		t.Fatalf("1º sync: gravados=%d tabela=%d, esperado 4/4", n, conta(empresaA))
	}

	// A data ausente tem que virar NULL, não erro de encoding nem data zerada.
	var comDataNula int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+schema+
		`.movimentos_financeiros WHERE empresa_id = $1 AND data_lancamento IS NULL`,
		empresaA).Scan(&comDataNula); err != nil {
		t.Fatalf("consulta data nula: %v", err)
	}
	if comDataNula != 1 {
		t.Errorf("esperado 1 registro com data_lancamento NULL, veio %d", comDataNula)
	}

	// Dados de outra empresa no mesmo schema — o schema é do grupo e abriga várias.
	if _, err := replaceMovimentos(ctx, pool, schema, empresaB, itens[:2], raws(2)); err != nil {
		t.Fatalf("sync empresa B: %v", err)
	}

	// 2º sync da empresa A: NÃO pode duplicar nem tocar na empresa B.
	if _, err := replaceMovimentos(ctx, pool, schema, empresaA, itens, raws(len(itens))); err != nil {
		t.Fatalf("2º sync: %v", err)
	}
	if got := conta(empresaA); got != 4 {
		t.Errorf("2º sync duplicou: empresa A tem %d linhas, esperado 4", got)
	}
	if got := conta(empresaB); got != 2 {
		t.Errorf("sync da empresa A afetou a empresa B: %d linhas, esperado 2", got)
	}

	// Resposta vazia não pode apagar o que existe.
	if _, err := replaceMovimentos(ctx, pool, schema, empresaA, nil, nil); err != nil {
		t.Fatalf("sync vazio: %v", err)
	}
	if got := conta(empresaA); got != 4 {
		t.Errorf("resposta vazia apagou dados: %d linhas, esperado 4", got)
	}

	// nCodTitulo = 0 vira NULL; valor cai para nValorMovCC quando nValorTitulo é 0.
	var semTitulo int
	var valor float64
	if err := pool.QueryRow(ctx, `SELECT count(*), max(valor) FROM `+schema+
		`.movimentos_financeiros WHERE empresa_id = $1 AND codigo_titulo IS NULL`,
		empresaA).Scan(&semTitulo, &valor); err != nil {
		t.Fatalf("consulta sem título: %v", err)
	}
	if semTitulo != 1 {
		t.Errorf("esperado 1 lançamento sem título, veio %d", semTitulo)
	}
	if valor != 5.5 {
		t.Errorf("valor do lançamento avulso deveria cair para nValorMovCC (5.5), veio %v", valor)
	}
}
