package ia

import (
	"encoding/json"
	"regexp"
	"strings"
)

/*
Extração do gráfico da resposta do modelo.

O modelo escreve texto em Markdown e, quando há gráfico, um bloco ```grafico com
JSON dentro. Separar os dois aqui — e não no navegador — tem duas razões:

  - o texto que vai para a tela sai limpo, sem um bloco de JSON cru no meio;
  - a spec é VALIDADA antes de virar gráfico. JSON malformado ou com série
    desalinhada dos rótulos produz gráfico errado, que é pior que gráfico
    nenhum: um gráfico errado parece certo.
*/

/*
blocoGrafico casa o bloco de especificação.

A versão anterior exigia exatamente "```grafico" seguido de quebra de linha, e o
modelo erra esse formato com facilidade: escreve "```json", "``` grafico",
"```GRAFICO", ou abre a chave na mesma linha. Em qualquer desses casos nada era
extraído e o JSON inteiro aparecia na tela como bloco de código — que é
exatamente o que o usuário relatou.

Aceita-se, então, grafico OU json como marcação, em qualquer caixa, com ou sem
espaço e com ou sem a quebra de linha. Um bloco ```json que não seja uma spec
válida continua sendo devolvido ao texto (ver ExtrairGrafico), porque nem todo
json numa resposta é um gráfico.
*/
var blocoGrafico = regexp.MustCompile("(?is)```[ \\t]*(?:grafico|gráfico|json)[ \\t]*\\r?\\n?(.*?)```")

// blocoAberto pega a marcação de abertura sem fechamento — o que sobra quando a
// resposta é cortada no limite de tokens no meio do bloco.
var blocoAberto = regexp.MustCompile("(?is)```[ \\t]*(?:grafico|gráfico|json)[ \\t]*\\r?\\n?[^`]*$")

/*
LimparBlocoAberto remove um bloco que começou e nunca fechou.

Só faz sentido quando a resposta veio truncada: o fechamento ficou do outro lado
do corte, nenhuma regex casa, e o JSON pela metade iria para a tela.
*/
func LimparBlocoAberto(texto string) string {
	return strings.TrimSpace(blocoAberto.ReplaceAllString(texto, ""))
}

// maxRotulos limita o que vira gráfico. Acima disso as legendas se sobrepõem e
// a figura deixa de comunicar — o próprio prompt já pede para agrupar.
const maxRotulos = 12

var tiposValidos = map[string]bool{
	"barra":            true,
	"barra_horizontal": true,
	"linha":            true,
	"rosca":            true,
	// "numero" é um card de indicador, não um gráfico: um valor só, em
	// destaque. Existe porque a resposta mais comum do financeiro é um número —
	// e desenhar uma barra sozinha para mostrá-lo é pior que escrevê-lo grande.
	"numero": true,
}

/*
ExtrairGrafico separa o texto da especificação.

Devolve o texto SEM nenhum bloco de código-dado e a spec, quando existe uma
válida. O terceiro retorno é o motivo do descarte — vazio quando não houve — e
existe porque o descarte era invisível: uma spec recusada sumia sem log e sem
aviso, e ninguém ficava sabendo que o modelo tinha tentado desenhar algo.

TODOS os blocos são removidos do texto, inclusive os inválidos. Um bloco de JSON
cru na tela não ajuda ninguém a entender a resposta — é o ruído que motivou esta
função. Quando há mais de um bloco, vale o primeiro que produzir spec válida.
*/
func ExtrairGrafico(resposta string) (texto string, spec *SpecGrafico, descarte string) {
	blocos := blocoGrafico.FindAllStringSubmatch(resposta, -1)
	if blocos == nil {
		return strings.TrimSpace(resposta), nil, ""
	}

	texto = strings.TrimSpace(blocoGrafico.ReplaceAllString(resposta, ""))

	for _, b := range blocos {
		var s SpecGrafico
		if err := json.Unmarshal([]byte(b[1]), &s); err != nil {
			descarte = "json malformado"
			continue
		}
		if motivo := motivoInvalido(&s); motivo != "" {
			descarte = motivo
			continue
		}
		return texto, &s, ""
	}
	return texto, nil, descarte
}

/*
motivoInvalido recusa o que não vira gráfico honesto, e diz por quê.

A checagem que mais importa é o alinhamento entre série e rótulos. O Chart.js
desenha uma série mais curta sem reclamar, casando pelo índice — e o resultado é
um gráfico em que a barra de março mostra o valor de abril. Ninguém percebe
olhando, e é um gráfico sobre dinheiro.

Devolve string vazia quando a spec serve. Também NORMALIZA: formato inválido
vira moeda, e rosca com várias séries perde as excedentes em vez de ser
descartada — ver abaixo.
*/
func motivoInvalido(s *SpecGrafico) string {
	if s == nil {
		return "spec vazia"
	}
	if !tiposValidos[s.Tipo] {
		return "tipo desconhecido: " + s.Tipo
	}
	if len(s.Rotulos) == 0 {
		return "sem rótulos"
	}
	if len(s.Rotulos) > maxRotulos {
		return "rótulos demais"
	}
	if len(s.Series) == 0 {
		return "sem séries"
	}
	for _, serie := range s.Series {
		if len(serie.Valores) != len(s.Rotulos) {
			return "série desalinhada dos rótulos"
		}
	}
	/*
	 * Rosca só desenha uma série. Antes a spec inteira era descartada e o
	 * usuário ficava sem gráfico nenhum; agora fica com a primeira série, que é
	 * o que ele veria de qualquer forma — e é melhor que nada.
	 */
	if s.Tipo == "rosca" && len(s.Series) > 1 {
		s.Series = s.Series[:1]
	}
	// Card de indicador é um valor só. Mais que isso é gráfico, e o tipo está
	// errado.
	if s.Tipo == "numero" && (len(s.Rotulos) != 1 || len(s.Series[0].Valores) != 1) {
		return "numero aceita um único valor"
	}
	if s.Formato != "moeda" && s.Formato != "numero" {
		s.Formato = "moeda"
	}
	return ""
}
