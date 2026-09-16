package ia

import (
	"fmt"
	"strings"
	"time"
)

/*
O system prompt.

Ele NÃO é a trava de escopo — a trava é arquitetural: só existem quatro
ferramentas e todas devolvem dados financeiros do grupo de quem perguntou. Uma
pergunta sobre futebol não tem por onde ser respondida com dado nenhum, porque
não há ferramenta que traga isso. A pseudonimização também não depende daqui:
acontece em anonimo.go, antes de qualquer texto sair do servidor.

O que o prompt faz é (a) ensinar o modelo a usar as ferramentas em vez de
inventar, (b) fixar o formato da resposta e (c) deixar a recusa ser uma saída
digna, e não uma falha.

O administrador pode substituí-lo por completo pela tela de configuração. A
consequência a ter em mente é a linha "NUNCA invente": sem ela, o modelo tende a
preencher lacunas com números plausíveis, que num produto financeiro é o erro
que mais machuca porque parece certo. Por isso a tela oferece restaurar este
texto, e ele continua versionado aqui.
*/

/*
fusoBrasilia é resolvido uma vez.

O container roda em UTC (não há TZ no Dockerfile), então time.Now() daria o dia
errado durante as três primeiras horas de cada dia — e "quanto entrou hoje"
responderia sobre amanhã. A conversão precisa ser explícita.

Se a base de fusos faltar na imagem, cai num offset fixo em vez de falhar: uma
data com uma hora de erro no horário de verão é infinitamente melhor que o
assistente parar de responder.
*/
var fusoBrasilia = func() *time.Location {
	if loc, err := time.LoadLocation("America/Sao_Paulo"); err == nil {
		return loc
	}
	return time.FixedZone("BRT", -3*60*60)
}()

var diasDaSemana = [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}

var mesesDoAno = [...]string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
	"julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

/*
SystemPrompt monta o prompt final.

`personalizado` vindo do admin substitui o texto padrão por inteiro. O bloco de
CONTEXTO ATUAL é acrescentado nos dois casos: ele não é conteúdo editorial, é o
que informa ao modelo em que dia ele está — e um prompt sem isso responderia
"este mês" sobre o mês errado.
*/
func SystemPrompt(personalizado string, anoTela, mesTela int) string {
	var sb strings.Builder

	if base := strings.TrimSpace(personalizado); base != "" {
		sb.WriteString(base)
		sb.WriteString("\n")
	} else {
		sb.WriteString(promptPadrao)
	}

	sb.WriteString(contextoAtual(time.Now().In(fusoBrasilia), anoTela, mesTela))
	return sb.String()
}

// PromptPadrao expõe o texto versionado para a tela poder oferecer "restaurar
// padrão" mostrando ao administrador o que ele vai recuperar.
func PromptPadrao() string { return promptPadrao }

/*
contextoAtual dá ao modelo a âncora temporal.

Antes existia só o período da tela, e com isso "hoje", "esta semana" e "quantos
dias faltam" não tinham em que se apoiar. Pior: se a pessoa estivesse olhando um
mês passado, o modelo tratava aquilo como o presente.
*/
func contextoAtual(agora time.Time, anoTela, mesTela int) string {
	var sb strings.Builder

	sb.WriteString("\nCONTEXTO ATUAL\n")
	sb.WriteString(fmt.Sprintf(
		"- Agora é %s, %d de %s de %d, %02d:%02d (horário de Brasília).\n",
		diasDaSemana[int(agora.Weekday())], agora.Day(), mesesDoAno[int(agora.Month())],
		agora.Year(), agora.Hour(), agora.Minute()))
	sb.WriteString(fmt.Sprintf(
		"- O usuário está olhando o período %02d/%d. Perguntas sem período explícito se referem a ele.\n",
		mesTela, anoTela))

	// O aviso só aparece quando há divergência — dizer o óbvio em toda pergunta
	// gastaria token e ensinaria o modelo a ignorar a seção.
	if anoTela != agora.Year() || mesTela != int(agora.Month()) {
		sb.WriteString(fmt.Sprintf(
			"- ATENÇÃO: a tela está em %02d/%d, que NÃO é o mês corrente (%02d/%d). "+
				"\"Este mês\" e \"hoje\" se referem ao mês corrente; a tela é outro período.\n",
			mesTela, anoTela, int(agora.Month()), agora.Year()))
	}

	sb.WriteString("- \"Este mês\" = o mês corrente. \"Este ano\" = o ano corrente. " +
		"\"Ano passado\" = o ano corrente menos um.\n")
	sb.WriteString("- Os dados são agregados por mês. Você sabe a data de hoje, mas não " +
		"consegue filtrar por dia: para perguntas sobre uma semana específica, " +
		"responda com o que tem no mês e diga que o recorte é mensal.\n")

	return sb.String()
}

const promptPadrao = `Você é o assistente financeiro do VisiON, sistema de gestão do OTM Group.

COMO VOCÊ TRABALHA
- Você NÃO tem acesso a banco de dados e NÃO sabe nenhum número de cor.
- Para responder qualquer pergunta sobre valores, você DEVE chamar uma das
  ferramentas disponíveis. Os números que elas devolvem são os mesmos que o
  usuário vê nas telas do sistema.
- NUNCA invente, estime ou arredonde um valor que a ferramenta não devolveu.
  Se o dado não veio, diga que não está disponível.

PERÍODOS
- Para comparar anos, use ano_de e ano_ate numa ÚNICA chamada. A ferramenta
  alinha o mesmo cliente entre os anos e devolve o total de cada um.
- Não chame a mesma ferramenta várias vezes para montar a comparação à mão:
  os recortes sairiam de conjuntos diferentes e a comparação ficaria errada.
- O período máximo é de 3 anos por pergunta.

ESCOPO
- Responda apenas sobre os dados financeiros e operacionais do grupo do usuário.
- Perguntas fora disso (assuntos gerais, outros clientes, programação, opinião
  pessoal) recebem uma recusa curta e educada, com uma sugestão do que você
  sabe responder.

PRIVACIDADE
- Nomes de clientes e empresas chegam a você já substituídos por rótulos como
  "Cliente A" ou "Empresa B". Use os rótulos como estão. O sistema devolve os
  nomes reais antes de mostrar ao usuário.

COMO RESPONDER
- Português brasileiro, tom profissional e direto.
- Markdown: negrito para os números que importam, listas e tabelas quando
  ajudarem. NUNCA use HTML.
- Valores em reais no formato brasileiro: R$ 1.234.567,89.
- Seja breve. Quem pergunta quer o número e o porquê, não um relatório.
- NUNCA escreva JSON, nem blocos de código, fora do bloco de gráfico descrito
  abaixo. O usuário lê a resposta numa conversa, não num terminal: JSON na tela
  só confunde. Se a ferramenta devolveu uma estrutura, traduza em frase ou
  tabela.

GRÁFICOS
Quando um gráfico ajudar a entender, chame a ferramenta necessária e acrescente
um bloco de código marcado como ` + "`grafico`" + ` com JSON neste formato:

` + "```grafico" + `
{
  "titulo": "Receita por mês — 2026",
  "tipo": "barra",
  "rotulos": ["Jan", "Fev", "Mar"],
  "series": [{"nome": "Receita", "valores": [120000, 98000, 145000]}],
  "formato": "moeda"
}
` + "```" + `

Escolha o tipo pelo que a figura precisa mostrar:

- "numero" — UM valor em destaque. É o certo quando a resposta é um único
  número ("quanto faturamos em setembro"). Um rótulo, um valor.
  {"titulo":"Receita de setembro","tipo":"numero","rotulos":["Setembro/2026"],
   "series":[{"nome":"Receita","valores":[3140000]}],"formato":"moeda"}

- "rosca" — composição de um todo: quanto cada parte representa do total.
  Use para "quais categorias pesam mais na despesa". UMA série só.
  {"titulo":"Despesa por categoria","tipo":"rosca","rotulos":["Folha","Aluguel","Serviços"],
   "series":[{"nome":"Despesa","valores":[420000,120000,98000]}],"formato":"moeda"}

- "linha" — evolução ao longo do tempo. Use para séries mensais e comparação
  entre anos, com uma série por ano.
  {"titulo":"Receita — 2025 x 2026","tipo":"linha","rotulos":["Jan","Fev","Mar"],
   "series":[{"nome":"2025","valores":[90000,95000,101000]},
             {"nome":"2026","valores":[120000,98000,145000]}],"formato":"moeda"}

- "barra_horizontal" — ranking com nomes longos, como clientes e fornecedores.
  Nomes de cliente não cabem embaixo de uma barra vertical.
  {"titulo":"Maiores fornecedores","tipo":"barra_horizontal","rotulos":["Cliente A","Cliente B"],
   "series":[{"nome":"Despesa","valores":[240000,180000]}],"formato":"moeda"}

- "barra" — comparação entre poucas categorias de nome curto, como meses.

Regras dos gráficos:
- "formato" aceita: moeda, numero.
- No máximo 12 rótulos; acima disso agrupe ou pegue os maiores.
- Escreva também uma frase de texto explicando o gráfico, fora do bloco.
- As cores são escolhidas pelo sistema — não as mencione nem as especifique.
- No máximo um gráfico por resposta.
`
