package db

import (
	"strings"
	"testing"
)

const schemaDeTeste = "grupo_teste"

// As views gerenciais são strings com dezenas de %s posicionais. Um placeholder a
// mais produz "%!s(MISSING)" e um a menos produz "%!(EXTRA string=...)" — nos dois
// casos o SQL sai sintaticamente quebrado ou apontando para o schema errado, e sem
// este teste o erro só apareceria num REFRESH contra o schema de um cliente.
func TestProvisionStatements_SemErroDeFormatacao(t *testing.T) {
	for i, stmt := range provisionStatements(schemaDeTeste) {
		for _, marcador := range []string{"%!s(MISSING)", "%!(EXTRA", "%!s(", "%!d("} {
			if strings.Contains(stmt, marcador) {
				trecho := stmt
				if len(trecho) > 300 {
					trecho = trecho[:300] + "..."
				}
				t.Errorf("statement %d contém %s (argumentos do Sprintf não batem com os placeholders):\n%s",
					i, marcador, trecho)
				break
			}
		}
	}
}

// Isolamento de tenant: nenhum statement pode referenciar um schema que não seja o
// do grupo, com a única exceção de _etl, que guarda as tabelas de controle.
func TestProvisionStatements_SomenteSchemaDoGrupoEEtl(t *testing.T) {
	stmts := provisionStatements(schemaDeTeste)
	if len(stmts) == 0 {
		t.Fatal("provisionStatements não devolveu nenhum statement")
	}

	for i, stmt := range stmts {
		for _, tabela := range []string{
			"contratos", "extrato", "movimentos_financeiros",
			"contas_pagar", "contas_receber", "clientes", "departamentos", "categorias",
		} {
			// Uma referência qualificada só é legítima se o qualificador for o schema
			// do grupo. "public.contratos" ou "outrogrupo.contratos" seriam vazamento.
			for _, ref := range referenciasQualificadas(stmt, tabela) {
				// Os blocos de auto-upgrade comparam pg_matviews.definition contra
				// padrões como '%.extrato %'. São literais de LIKE, não referências
				// a schema — um identificador real nunca contém % nem aspas simples.
				if strings.ContainsAny(ref, "%'") {
					continue
				}
				if ref != schemaDeTeste && ref != "_etl" {
					t.Errorf("statement %d referencia %s.%s, fora do schema do grupo", i, ref, tabela)
				}
			}
		}
	}
}

// As duas CTEs de contrato precisam existir nas DUAS matviews: o ramo mov, que
// também recebe o fallback via detalhes.cNumCtr, existe em ambas. Se uma delas
// perder as CTEs, o fallback some em silêncio para metade do período.
func TestProvisionStatements_ContratosNasDuasMatviews(t *testing.T) {
	encontradas := map[string]bool{}

	for _, stmt := range provisionStatements(schemaDeTeste) {
		for _, view := range []string{"matvw_gerencial_ano_corrente", "matvw_gerencial_historico"} {
			if !strings.Contains(stmt, "CREATE MATERIALIZED VIEW IF NOT EXISTS "+qualificado(view)) {
				continue
			}
			encontradas[view] = true

			for _, exigido := range []string{
				"ctr_cabecalho AS (",
				"ctr_distribuicao AS (",
				"LEFT JOIN ctr_cabecalho",
				"LEFT JOIN ctr_distribuicao",
				// O contrato nunca pode sobrescrever um título de CP/CR.
				"c.origem IS NULL",
				// Guard obrigatório: contrato sem rateio traz departamentos ausente
				// ou como objeto, e jsonb_array_elements sobre não-array levanta
				// exceção, derrubando o REFRESH inteiro.
				"jsonb_typeof(ctr.raw -> 'departamentos') = 'array'",
				// Deduplicação: sem ela, contrato com dois departamentos duplicaria
				// o valor do lançamento e inflaria os totais da view.
				"SELECT DISTINCT ON (empresa_id, numero_contrato)",
			} {
				if !strings.Contains(stmt, exigido) {
					t.Errorf("%s não contém %q", view, exigido)
				}
			}

			// Contagem, não presença: cada referência de contrato aparece em pontos
			// distintos da view (coluna derivada, CASE do rótulo, condição do join),
			// e verificar só "contém" deixaria passar a remoção de uma delas — o
			// fallback sumiria num dos pontos sem nenhum teste reclamar.
			for expr, esperado := range map[string]int{
				// codigo_departamento_join, o CASE de departamento_final e o join
				// com a tabela departamentos.
				"ctrd.codigo_departamento": 3,
				// cod_categoria_final_superior, cod_categoria_final e os joins
				// cat_sup e cat_final.
				"ctrc.codigo_categoria": 4,
				// Só o join com clientes.
				"ctrc.codigo_cliente": 1,
			} {
				if got := strings.Count(stmt, expr); got != esperado {
					t.Errorf("%s: %q aparece %d vez(es), esperado %d", view, expr, got, esperado)
				}
			}

			// Precedência do departamento: título, depois contrato, depois movimento.
			if !strings.Contains(stmt, "COALESCE(d.codigo_departamento, ctrd.codigo_departamento, m.departamento_mov)") {
				t.Errorf("%s: precedência de departamento não inclui o contrato na posição correta", view)
			}
		}
	}

	for _, view := range []string{"matvw_gerencial_ano_corrente", "matvw_gerencial_historico"} {
		if !encontradas[view] {
			t.Errorf("statement de CREATE da %s não foi encontrado", view)
		}
	}
}

// O ramo ext casa por cNumero; o ramo mov, por cNumCtr. São campos diferentes de
// APIs diferentes e trocá-los faria o join falhar sem erro.
func TestProvisionStatements_ChavesDeContratoPorRamo(t *testing.T) {
	var anoCorrente string
	for _, stmt := range provisionStatements(schemaDeTeste) {
		if strings.Contains(stmt, "CREATE MATERIALIZED VIEW IF NOT EXISTS "+qualificado("matvw_gerencial_ano_corrente")) {
			anoCorrente = stmt
		}
	}
	if anoCorrente == "" {
		t.Fatal("matvw_gerencial_ano_corrente não encontrada")
	}
	if !strings.Contains(anoCorrente, "e.raw ->> 'cNumero'                    AS numero_contrato") {
		t.Error("ramo ext não projeta cNumero como numero_contrato")
	}
	if !strings.Contains(anoCorrente, "mf.raw -> 'detalhes' ->> 'cNumCtr'") {
		t.Error("ramo mov não projeta detalhes.cNumCtr como numero_contrato")
	}
}

func qualificado(nome string) string {
	return `"` + schemaDeTeste + `".` + nome
}

// referenciasQualificadas devolve os qualificadores usados antes de ".<tabela>",
// aceitando tanto a forma sanitizada ("grupo"."tabela") quanto a nua.
func referenciasQualificadas(stmt, tabela string) []string {
	var out []string
	for _, sufixo := range []string{`."` + tabela, "." + tabela} {
		resto := stmt
		for {
			i := strings.Index(resto, sufixo)
			if i < 0 {
				break
			}
			out = append(out, qualificador(resto[:i]))
			resto = resto[i+len(sufixo):]
		}
	}
	return out
}

func qualificador(antes string) string {
	antes = strings.TrimSuffix(antes, `"`)
	corte := strings.LastIndexAny(antes, " \t\n(,")
	ident := antes[corte+1:]
	return strings.Trim(ident, `"`)
}
