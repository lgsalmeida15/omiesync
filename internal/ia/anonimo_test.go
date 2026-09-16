package ia

import (
	"strings"
	"testing"
)

func TestAnonimizador_IdaEVolta(t *testing.T) {
	a := NovoAnonimizador()

	r := a.Rotular("Cliente", "Constrimax LTDA")
	if r == "Constrimax LTDA" {
		t.Fatal("o nome real saiu sem rótulo")
	}

	texto := "O maior faturamento veio de " + r + "."
	if got := a.Restaurar(texto); got != "O maior faturamento veio de Constrimax LTDA." {
		t.Fatalf("got %q", got)
	}
}

/*
O mesmo cliente precisa receber o mesmo rótulo em toda a conversa. Sem isso a IA
enxerga dois clientes onde há um, e soma errado.
*/
func TestAnonimizador_MesmoNomeMesmoRotulo(t *testing.T) {
	a := NovoAnonimizador()

	primeiro := a.Rotular("Cliente", "Delta Serviços")
	segundo := a.Rotular("Cliente", "Delta Serviços")
	comEspaco := a.Rotular("Cliente", "  Delta Serviços  ")

	if primeiro != segundo || primeiro != comEspaco {
		t.Fatalf("rótulos divergiram: %q, %q, %q", primeiro, segundo, comEspaco)
	}
	if a.Total() != 1 {
		t.Fatalf("criou %d entradas para um cliente só", a.Total())
	}
}

func TestAnonimizador_NomesDiferentesRotulosDiferentes(t *testing.T) {
	a := NovoAnonimizador()
	vistos := map[string]bool{}

	for _, nome := range []string{"Alpha", "Beta", "Gama", "Delta"} {
		r := a.Rotular("Cliente", nome)
		if vistos[r] {
			t.Fatalf("rótulo repetido: %q", r)
		}
		vistos[r] = true
	}
}

// Prefixos distintos não podem colidir: um "Cliente A" e uma "Empresa A"
// coexistem, e restaurar um não pode arrastar o outro.
func TestAnonimizador_PrefixosIndependentes(t *testing.T) {
	a := NovoAnonimizador()
	cli := a.Rotular("Cliente", "Constrimax")
	emp := a.Rotular("Empresa", "Alpha Comércio")

	if cli == emp {
		t.Fatalf("prefixos colidiram: ambos %q", cli)
	}
	got := a.Restaurar(cli + " e " + emp)
	if got != "Constrimax e Alpha Comércio" {
		t.Fatalf("got %q", got)
	}
}

/*
O caso que a ordem de substituição existe para resolver.

Com 27 clientes existem "Cliente A" e "Cliente AA". Substituindo do mais curto
para o mais longo, "Cliente A" casaria DENTRO de "Cliente AA" e produziria
"<nome de A>A" — um nome que não existe, numa resposta sobre dinheiro.
*/
func TestAnonimizador_RotuloLongoNaoEComidoPeloCurto(t *testing.T) {
	a := NovoAnonimizador()

	var primeiro, vigesimoSetimo string
	for i := 1; i <= 27; i++ {
		nome := "Empresa número " + string(rune('a'+i%26)) + string(rune('0'+i/10))
		r := a.Rotular("Cliente", nome)
		if i == 1 {
			primeiro = r
		}
		if i == 27 {
			vigesimoSetimo = r
		}
	}

	if primeiro != "Cliente A" || vigesimoSetimo != "Cliente AA" {
		t.Fatalf("a numeração não chegou ao caso interessante: %q e %q", primeiro, vigesimoSetimo)
	}

	got := a.Restaurar("Comparando " + vigesimoSetimo + " com " + primeiro + ".")
	if strings.Contains(got, "Cliente") {
		t.Fatalf("sobrou rótulo sem restaurar: %q", got)
	}
	// O nome do 27º tem de aparecer inteiro, não o do 1º seguido de "A".
	if !strings.Contains(got, a.paraReal["Cliente AA"]) {
		t.Fatalf("o rótulo longo foi comido pelo curto: %q", got)
	}
}

// "Não informado" já chega assim do banco; rotular isso criaria um cliente
// fantasma que a IA trataria como entidade real.
func TestAnonimizador_NomeVazioNaoVirarRotulo(t *testing.T) {
	a := NovoAnonimizador()
	for _, vazio := range []string{"", "   "} {
		if r := a.Rotular("Cliente", vazio); r != "" {
			t.Fatalf("nome vazio virou %q", r)
		}
	}
	if a.Total() != 0 {
		t.Fatalf("criou %d entradas para nada", a.Total())
	}
}

// Modelo alucina. Um rótulo que não existe no mapa não pode derrubar a resposta
// inteira — é melhor a tela mostrar "Cliente Z" do que não mostrar nada.
func TestAnonimizador_RotuloDesconhecidoNaoQuebra(t *testing.T) {
	a := NovoAnonimizador()
	a.Rotular("Cliente", "Constrimax")

	got := a.Restaurar("Comparei Cliente A com Cliente Z.")
	if !strings.Contains(got, "Constrimax") {
		t.Fatalf("o rótulo conhecido não foi restaurado: %q", got)
	}
	if !strings.Contains(got, "Cliente Z") {
		t.Fatalf("o rótulo inventado deveria passar intacto: %q", got)
	}
}

func TestAnonimizador_SemMapaTextoIntacto(t *testing.T) {
	a := NovoAnonimizador()
	const texto = "Receita de setembro: R$ 6,07 mi."
	if got := a.Restaurar(texto); got != texto {
		t.Fatalf("got %q", got)
	}
}

func TestSufixo(t *testing.T) {
	casos := map[int]string{1: "A", 2: "B", 26: "Z", 27: "AA", 28: "AB", 52: "AZ", 53: "BA", 703: "AAA"}
	for n, esperado := range casos {
		if got := sufixo(n); got != esperado {
			t.Errorf("sufixo(%d) = %q, esperava %q", n, got, esperado)
		}
	}
	// Nunca devolver string vazia: um rótulo vazio viraria "Cliente " e
	// casaria com tudo na restauração.
	if sufixo(0) == "" || sufixo(-1) == "" {
		t.Fatal("sufixo devolveu vazio")
	}
}

/*
A garantia que importa: NENHUM nome real pode sobrar no texto que sai daqui.
Percorre todos os nomes rotulados e confirma que nenhum aparece.
*/
func TestAnonimizador_NenhumNomeRealVazaNaSaida(t *testing.T) {
	a := NovoAnonimizador()
	nomes := []string{"Constrimax LTDA", "Delta Serviços ME", "Alpha Comércio S/A"}

	var partes []string
	for _, n := range nomes {
		partes = append(partes, a.Rotular("Cliente", n))
	}
	enviado := "Top 3: " + strings.Join(partes, ", ") + "."

	for _, n := range nomes {
		if strings.Contains(enviado, n) {
			t.Fatalf("o nome %q foi para o provedor em claro: %q", n, enviado)
		}
	}
}

/*
O furo que este método fecha.

A resposta é gravada no histórico com os nomes REAIS — é o que a tela precisa
mostrar. Esse mesmo texto volta ao provedor como contexto na pergunta seguinte.
Sem Ocultar, a pseudonimização protegia só a primeira pergunta de cada conversa,
e da segunda em diante o nome do cliente ia em claro para fora.
*/
func TestOcultar_HistoricoNaoVazaNomeReal(t *testing.T) {
	a := NovoAnonimizador()
	rotulo := a.Rotular("Cliente", "Constrimax LTDA")

	// Simula uma resposta já gravada, com o nome restaurado.
	gravado := "O maior faturamento veio de **Constrimax LTDA**, com R$ 1,2 mi."

	saida := a.Ocultar(gravado)

	if strings.Contains(saida, "Constrimax") {
		t.Fatalf("o nome real sobreviveu e iria para o provedor: %q", saida)
	}
	if !strings.Contains(saida, rotulo) {
		t.Fatalf("não colocou o rótulo no lugar: %q", saida)
	}
	// O resto do texto tem de continuar inteiro, senão o contexto se perde.
	if !strings.Contains(saida, "R$ 1,2 mi") {
		t.Fatalf("estragou o texto em volta: %q", saida)
	}
}

// Ida e volta: ocultar o que foi restaurado devolve exatamente o rótulo de
// origem. É o que mantém um vocabulário só na conversa inteira.
func TestOcultar_EhOInversoDeRestaurar(t *testing.T) {
	a := NovoAnonimizador()
	a.Rotular("Cliente", "Alpha Comercio")

	original := "Cliente A cresceu."
	restaurado := a.Restaurar(original)
	if restaurado == original {
		t.Fatal("Restaurar não fez nada — teste inválido")
	}

	if volta := a.Ocultar(restaurado); volta != original {
		t.Fatalf("Ocultar(Restaurar(x)) = %q, esperava %q", volta, original)
	}
}

/*
Nome curto contido em nome longo.

"Alpha" está dentro de "Alpha Comercio". Sem ordenar do mais longo para o mais
curto, "Alpha Comercio" viraria "Cliente B Comercio" — um nome que não existe,
numa conversa sobre dinheiro. É o mesmo cuidado que Restaurar já tomava, na
outra direção.
*/
func TestOcultar_NomeCurtoDentroDeNomeLongo(t *testing.T) {
	a := NovoAnonimizador()
	rotuloLongo := a.Rotular("Cliente", "Alpha Comercio")
	a.Rotular("Cliente", "Alpha")

	saida := a.Ocultar("Faturamento de Alpha Comercio no mês.")

	if !strings.Contains(saida, rotuloLongo) {
		t.Fatalf("não casou o nome longo: %q", saida)
	}
	if strings.Contains(saida, "Comercio") {
		t.Fatalf("partiu o nome ao meio: %q", saida)
	}
}

// Nome que nunca foi rotulado fica como está: é o que torna a pré-população
// (Executor.PrepararRotulos) parte da proteção, e não um detalhe de desempenho.
func TestOcultar_NomeDesconhecidoNaoMuda(t *testing.T) {
	a := NovoAnonimizador()
	a.Rotular("Cliente", "Alpha Comercio")

	texto := "Beta Servicos apareceu."
	if saida := a.Ocultar(texto); saida != texto {
		t.Fatalf("mexeu num nome que não conhece: %q", saida)
	}
}
