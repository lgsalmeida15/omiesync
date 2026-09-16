package ia

import (
	"strings"
	"testing"
)

const blocoOK = "```grafico\n" + `{
  "titulo": "Receita por mês",
  "tipo": "barra",
  "rotulos": ["Jan","Fev","Mar"],
  "series": [{"nome":"Receita","valores":[100,200,300]}],
  "formato": "moeda"
}` + "\n```"

func TestExtrairGrafico_SeparaTextoDaSpec(t *testing.T) {
	resposta := "A receita cresceu no trimestre.\n\n" + blocoOK + "\n\nMarço foi o melhor mês."

	texto, spec, _ := ExtrairGrafico(resposta)

	if spec == nil {
		t.Fatal("não extraiu a spec")
	}
	if spec.Titulo != "Receita por mês" || spec.Tipo != "barra" {
		t.Fatalf("spec errada: %+v", spec)
	}
	// O bloco não pode sobrar no texto — apareceria como JSON cru na bolha.
	if strings.Contains(texto, "```") || strings.Contains(texto, "rotulos") {
		t.Fatalf("o bloco vazou para o texto: %q", texto)
	}
	if !strings.Contains(texto, "A receita cresceu") || !strings.Contains(texto, "melhor mês") {
		t.Fatalf("perdeu texto: %q", texto)
	}
}

func TestExtrairGrafico_SemBloco(t *testing.T) {
	texto, spec, _ := ExtrairGrafico("  A receita de setembro foi R$ 6,07 mi.  ")
	if spec != nil {
		t.Fatal("inventou uma spec")
	}
	if texto != "A receita de setembro foi R$ 6,07 mi." {
		t.Fatalf("got %q", texto)
	}
}

/*
Bloco quebrado não pode custar a resposta ao usuário.

O modelo erra o JSON com alguma frequência. Quando isso acontece a explicação em
texto ainda vale — melhor perder o gráfico que perder tudo.
*/
func TestExtrairGrafico_JSONQuebradoPreservaOTexto(t *testing.T) {
	resposta := "Segue a comparação.\n\n```grafico\n{isso não é json}\n```"

	texto, spec, _ := ExtrairGrafico(resposta)
	if spec != nil {
		t.Fatal("aceitou JSON quebrado")
	}
	if !strings.Contains(texto, "Segue a comparação") {
		t.Fatalf("perdeu o texto: %q", texto)
	}
	if strings.Contains(texto, "```") {
		t.Fatalf("sobrou o bloco: %q", texto)
	}
}

/*
O defeito que este arquivo existe para impedir.

Série com mais ou menos valores que rótulos: o Chart.js desenha assim mesmo,
alinhando pelo índice, e a barra de março passa a mostrar o valor de abril.
Ninguém percebe olhando — e é um gráfico sobre dinheiro.
*/
func TestSpecValida_SerieDesalinhadaDosRotulos(t *testing.T) {
	casos := []struct {
		nome    string
		rotulos []string
		valores []float64
	}{
		{"valores a mais", []string{"Jan", "Fev"}, []float64{1, 2, 3}},
		{"valores a menos", []string{"Jan", "Fev", "Mar"}, []float64{1, 2}},
		{"sem valores", []string{"Jan"}, []float64{}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := &SpecGrafico{
				Tipo: "barra", Rotulos: c.rotulos, Formato: "moeda",
				Series: []SerieGrafico{{Nome: "x", Valores: c.valores}},
			}
			if motivoInvalido(s) == "" {
				t.Fatalf("aceitou %d rótulos com %d valores", len(c.rotulos), len(c.valores))
			}
		})
	}
}

func TestSpecValida_Recusas(t *testing.T) {
	base := func() *SpecGrafico {
		return &SpecGrafico{
			Tipo: "barra", Rotulos: []string{"Jan"}, Formato: "moeda",
			Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
		}
	}

	t.Run("tipo desconhecido", func(t *testing.T) {
		s := base()
		s.Tipo = "pizza3d"
		if motivoInvalido(s) == "" {
			t.Fatal("aceitou tipo que o frontend não sabe desenhar")
		}
	})

	t.Run("sem rótulos", func(t *testing.T) {
		s := base()
		s.Rotulos = nil
		if motivoInvalido(s) == "" {
			t.Fatal("aceitou gráfico sem eixo")
		}
	})

	t.Run("sem séries", func(t *testing.T) {
		s := base()
		s.Series = nil
		if motivoInvalido(s) == "" {
			t.Fatal("aceitou gráfico sem dado")
		}
	})

	// Acima de 12 as legendas se sobrepõem e a figura deixa de comunicar.
	t.Run("rótulos demais", func(t *testing.T) {
		s := base()
		s.Rotulos = make([]string, maxRotulos+1)
		s.Series[0].Valores = make([]float64, maxRotulos+1)
		if motivoInvalido(s) == "" {
			t.Fatalf("aceitou %d rótulos", maxRotulos+1)
		}
	})

	t.Run("nil", func(t *testing.T) {
		if motivoInvalido(nil) == "" {
			t.Fatal("aceitou nil")
		}
	})
}

// Formato desconhecido não invalida o gráfico — cai em moeda, que é o caso
// dominante num sistema financeiro.
func TestSpecValida_FormatoDesconhecidoCaiEmMoeda(t *testing.T) {
	s := &SpecGrafico{
		Tipo: "linha", Rotulos: []string{"Jan"}, Formato: "percentual",
		Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
	}
	if motivoInvalido(s) != "" {
		t.Fatal("recusou por causa do formato")
	}
	if s.Formato != "moeda" {
		t.Fatalf("formato = %q, esperava moeda", s.Formato)
	}
}

func TestSpecValida_TodosOsTiposAceitos(t *testing.T) {
	for tipo := range tiposValidos {
		s := &SpecGrafico{
			Tipo: tipo, Rotulos: []string{"Jan"}, Formato: "moeda",
			Series: []SerieGrafico{{Nome: "x", Valores: []float64{1}}},
		}
		if motivo := motivoInvalido(s); motivo != "" {
			t.Errorf("recusou o tipo %q, que o catálogo anuncia: %s", tipo, motivo)
		}
	}
}

/*
Rosca com várias séries: antes a spec inteira era descartada e o usuário ficava
sem gráfico nenhum. Agora fica com a primeira série — que é o que ele veria de
qualquer jeito, já que o Chart.js só desenha uma.
*/
func TestSpecValida_RoscaMultiSerieFicaComAPrimeira(t *testing.T) {
	s := &SpecGrafico{
		Tipo: "rosca", Rotulos: []string{"Jan"}, Formato: "moeda",
		Series: []SerieGrafico{
			{Nome: "receita", Valores: []float64{1}},
			{Nome: "despesa", Valores: []float64{2}},
		},
	}
	if motivo := motivoInvalido(s); motivo != "" {
		t.Fatalf("descartou em vez de normalizar: %s", motivo)
	}
	if len(s.Series) != 1 || s.Series[0].Nome != "receita" {
		t.Fatalf("séries = %+v, esperava só a primeira", s.Series)
	}
}

// O card de indicador é um valor só. Mais que isso é gráfico, e o tipo está
// errado — desenhar assim mesmo mostraria um número escondendo os outros.
func TestSpecValida_NumeroAceitaUmValorSo(t *testing.T) {
	s := &SpecGrafico{
		Tipo: "numero", Rotulos: []string{"Jan", "Fev"}, Formato: "moeda",
		Series: []SerieGrafico{{Nome: "x", Valores: []float64{1, 2}}},
	}
	if motivoInvalido(s) == "" {
		t.Fatal("aceitou card de indicador com dois valores")
	}
}

/*
As variações de marcação que o modelo realmente produz.

A regex antiga exigia exatamente "```grafico" seguido de quebra de linha. Qualquer
desvio — e o modelo desvia — fazia o JSON inteiro aparecer na tela como bloco de
código. Foi o que o usuário relatou.
*/
func TestExtrairGrafico_ToleraVariacoesDeMarcacao(t *testing.T) {
	corpo := `{"titulo":"T","tipo":"barra","rotulos":["Jan"],` +
		`"series":[{"nome":"Receita","valores":[100]}],"formato":"moeda"}`

	casos := map[string]string{
		"json em vez de grafico": "```json\n" + corpo + "\n```",
		"caixa alta":             "```GRAFICO\n" + corpo + "\n```",
		"espaço antes do nome":   "``` grafico\n" + corpo + "\n```",
		"sem quebra de linha":    "```grafico " + corpo + "```",
		"com acento":             "```gráfico\n" + corpo + "\n```",
	}

	for nome, bloco := range casos {
		t.Run(nome, func(t *testing.T) {
			texto, spec, _ := ExtrairGrafico("Veja o gráfico.\n\n" + bloco)
			if spec == nil {
				t.Fatalf("não extraiu a spec de %q", bloco)
			}
			if strings.Contains(texto, "```") || strings.Contains(texto, "rotulos") {
				t.Fatalf("o JSON vazou para a tela: %q", texto)
			}
		})
	}
}

/*
Bloco aberto e nunca fechado: o que sobra quando o provedor corta a resposta no
limite de tokens bem no meio do JSON. Sem tratamento, esse pedaço ia inteiro
para a bolha.
*/
func TestLimparBlocoAberto(t *testing.T) {
	truncada := "A maior despesa foi em Folha.\n\n```grafico\n{\"titulo\":\"Desp"

	limpo := LimparBlocoAberto(truncada)

	if strings.Contains(limpo, "```") || strings.Contains(limpo, "titulo") {
		t.Fatalf("sobrou o bloco cortado: %q", limpo)
	}
	if !strings.Contains(limpo, "A maior despesa foi em Folha.") {
		t.Fatalf("perdeu o texto que tinha chegado inteiro: %q", limpo)
	}
}

// Um bloco recusado precisa dizer por quê: o descarte silencioso escondia que o
// modelo tinha tentado desenhar e falhado.
func TestExtrairGrafico_InformaOMotivoDoDescarte(t *testing.T) {
	_, spec, motivo := ExtrairGrafico("Veja.\n\n```grafico\n{não é json}\n```")
	if spec != nil {
		t.Fatal("aceitou JSON quebrado")
	}
	if motivo == "" {
		t.Fatal("descartou em silêncio")
	}
}
