package ia

import (
	"reflect"
	"strings"
	"testing"
)

/*
Resolução do intervalo de anos.

O caso sem ano_de/ano_ate precisa continuar idêntico ao de antes: a imensa
maioria das perguntas é sobre um ano só, e qualquer mudança ali seria regressão
disfarçada de recurso.
*/
func TestAnos_ResolveOIntervalo(t *testing.T) {
	casos := []struct {
		nome     string
		args     argsFerramenta
		esperado []int
		cortado  bool
	}{
		{"sem intervalo usa o ano de referência", argsFerramenta{Ano: 2026}, []int{2026}, false},
		{"intervalo completo", argsFerramenta{Ano: 2026, AnoDe: 2024, AnoAte: 2026}, []int{2024, 2025, 2026}, false},
		{"do ano passado pra cá", argsFerramenta{Ano: 2026, AnoDe: 2025, AnoAte: 2026}, []int{2025, 2026}, false},
		// Só um lado do intervalo: o outro é o ano de referência da tela.
		{"só ano_de", argsFerramenta{Ano: 2026, AnoDe: 2025}, []int{2025, 2026}, false},
		{"só ano_ate", argsFerramenta{Ano: 2024, AnoAte: 2026}, []int{2024, 2025, 2026}, false},
		// Modelo trocando a ordem não deve custar a resposta.
		{"invertido é corrigido", argsFerramenta{Ano: 2026, AnoDe: 2026, AnoAte: 2024}, []int{2024, 2025, 2026}, false},
		{"mesmo ano nos dois lados", argsFerramenta{Ano: 2026, AnoDe: 2025, AnoAte: 2025}, []int{2025}, false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			anos, cortado := c.args.anos()
			if !reflect.DeepEqual(anos, c.esperado) {
				t.Fatalf("anos = %v, esperava %v", anos, c.esperado)
			}
			if cortado != c.cortado {
				t.Fatalf("cortado = %v, esperava %v", cortado, c.cortado)
			}
		})
	}
}

/*
O corte mantém os anos MAIS RECENTES.

Quem pede "dos últimos dez anos pra cá" quer chegar até hoje. Cortar pelo fim
devolveria 2017-2019 para quem perguntou sobre 2026 — uma resposta sobre outro
assunto, entregue com confiança.
*/
func TestAnos_CorteMantemOsMaisRecentes(t *testing.T) {
	anos, cortado := argsFerramenta{Ano: 2026, AnoDe: 2017, AnoAte: 2026}.anos()

	if !cortado {
		t.Fatal("não sinalizou o corte")
	}
	if len(anos) != maxAnosIntervalo {
		t.Fatalf("devolveu %d anos, o teto é %d", len(anos), maxAnosIntervalo)
	}
	if anos[len(anos)-1] != 2026 {
		t.Fatalf("perdeu o ano mais recente: %v", anos)
	}
}

// Um corte silencioso numa comparação vira conclusão errada com cara de certa.
// O aviso precisa chegar ao modelo dizendo qual período de fato foi usado.
func TestAvisoCorte_DeclaraOPeriodoUsado(t *testing.T) {
	aviso := avisoCorte([]int{2024, 2025, 2026})

	if aviso == "" {
		t.Fatal("cortou sem avisar")
	}
	for _, esperado := range []string{"2024", "2026", "3"} {
		if !strings.Contains(aviso, esperado) {
			t.Errorf("o aviso não menciona %q: %q", esperado, aviso)
		}
	}
}

// A fonte mostrada na tela precisa refletir o período realmente consultado.
func TestFiltrosPeriodo(t *testing.T) {
	t.Run("ano único", func(t *testing.T) {
		f := filtrosPeriodo([]int{2026})
		if f["ano"] != 2026 {
			t.Fatalf("filtros = %v", f)
		}
		if _, tem := f["ano_de"]; tem {
			t.Fatal("inventou intervalo onde há um ano só")
		}
	})

	t.Run("intervalo", func(t *testing.T) {
		f := filtrosPeriodo([]int{2024, 2025, 2026})
		if f["ano_de"] != 2024 || f["ano_ate"] != 2026 {
			t.Fatalf("filtros = %v", f)
		}
	})
}

/*
A procedência do número.

"opcoes_de_filtro" só lista o que existe e não produz valor nenhum. Se o modelo
a chamasse por último, a tela diria que o número veio dela — e quem fosse
conferir não acharia nada.
*/
func TestFonteResultante(t *testing.T) {
	t.Run("sem consultas não há fonte", func(t *testing.T) {
		if fonteResultante(nil) != nil {
			t.Fatal("inventou uma fonte")
		}
	})

	t.Run("prefere a consulta que trouxe números", func(t *testing.T) {
		f := fonteResultante([]Fonte{
			{Ferramenta: "abrir_por_categoria_ou_cliente"},
			{Ferramenta: "opcoes_de_filtro"},
		})
		if f == nil || f.Ferramenta != "abrir_por_categoria_ou_cliente" {
			t.Fatalf("fonte = %+v", f)
		}
	})

	t.Run("entre duas substantivas fica a última", func(t *testing.T) {
		f := fonteResultante([]Fonte{
			{Ferramenta: "resumo_financeiro"},
			{Ferramenta: "fluxo_do_mes"},
		})
		if f == nil || f.Ferramenta != "fluxo_do_mes" {
			t.Fatalf("fonte = %+v", f)
		}
	})

	// Se só houve consulta auxiliar, mostrar ela é melhor que não mostrar nada.
	t.Run("só auxiliares devolve a auxiliar", func(t *testing.T) {
		f := fonteResultante([]Fonte{{Ferramenta: "opcoes_de_filtro"}})
		if f == nil || f.Ferramenta != "opcoes_de_filtro" {
			t.Fatalf("fonte = %+v", f)
		}
	})
}
