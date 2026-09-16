package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/dados"
)

/*
As ferramentas que a IA pode chamar.

Este arquivo é a fronteira do produto. A IA não escreve SQL, não vê o banco e
não conhece nome de tabela: ela escolhe uma destas quatro funções e os filtros.
Quem calcula é o MESMO código que desenha o dashboard — e é daí que vem a
propriedade que faz o recurso valer alguma coisa: o chat não CONSEGUE
contradizer a tela.

É também onde o escopo restrito para de depender do system prompt. Não existe
ferramenta para nada além dos dados financeiros do grupo, então não há o que a
IA consulte fora disso, por mais criativa que seja a pergunta.
*/

/*
timeoutFerramenta corta a consulta antes que o servidor corte a resposta.

A rota do chat tem orçamento próprio (orcamentoPergunta, no handler), maior que
o WriteTimeout global de 15s. Ainda assim a consulta precisa de teto próprio:
sem ele, uma query lenta consome o orçamento inteiro e não sobra tempo para a
chamada ao modelo nem para uma mensagem honesta de "demorou demais".

Os endpoints analíticos não têm limite próprio: /pivot devolve o ano inteiro.
*/
const timeoutFerramenta = 8 * time.Second

// maxLinhasPivot limita o que vai para o modelo. Um plano de contas real
// facilmente passa de mil combinações categoria×cliente, e cada linha é token
// pago. Acima disso a resposta é truncada e a IA é avisada disso.
const maxLinhasPivot = 150

/*
maxAnosIntervalo limita o quanto uma pergunta pode olhar para trás.

Cada ano é uma consulta ao banco e um bloco de números no prompt. Três cobre "do
ano passado pra cá" e "nos últimos três anos", que são os pedidos reais, sem que
uma pergunta solta dispare dez consultas e estoure o orçamento de tempo.
*/
const maxAnosIntervalo = 3

// maxNomesPreCarregados teto para a pré-população de rótulos. É uma consulta a
// mais por pergunta; sem teto, um grupo com milhares de clientes pagaria caro
// por ela.
const maxNomesPreCarregados = 300

type Ferramenta struct {
	Nome      string
	Descricao string
	// Parametros é o JSON Schema que o provedor recebe.
	Parametros map[string]any
}

// Catalogo é o que se oferece ao modelo. Ordem estável para o prompt não mudar
// de uma chamada para outra sem motivo.
func Catalogo() []Ferramenta {
	return []Ferramenta{
		{
			Nome: "resumo_financeiro",
			Descricao: "Totais de receita, despesa, resultado e saldo em contas, mais a " +
				"série mês a mês. Use para desempenho geral, comparação entre meses e " +
				"evolução ao longo do tempo. Para comparar anos, informe ano_de e ano_ate " +
				"numa única chamada.",
			Parametros: esquema(map[string]any{
				"ano":     anoProp("Ano de referência. Padrão: o da tela."),
				"ano_de":  anoProp("Primeiro ano do período, para comparar vários anos."),
				"ano_ate": anoProp("Último ano do período, para comparar vários anos."),
			}, nil),
		},
		{
			Nome: "abrir_por_categoria_ou_cliente",
			Descricao: "Abre receitas e despesas por categoria e por cliente/fornecedor. " +
				"Use para 'quais clientes', 'quais fornecedores', 'quais categorias', " +
				"rankings e composição. Com um ano, traz os doze meses de cada linha; " +
				"com ano_de e ano_ate, traz o total de cada ano lado a lado, já alinhado " +
				"pelo mesmo cliente — use assim para comparar evolução entre anos.",
			Parametros: esquema(map[string]any{
				"ano":     anoProp("Ano de referência. Padrão: o da tela."),
				"ano_de":  anoProp("Primeiro ano do período, para comparar vários anos."),
				"ano_ate": anoProp("Último ano do período, para comparar vários anos."),
			}, nil),
		},
		{
			Nome: "fluxo_do_mes",
			Descricao: "Lançamentos de um mês, com o que foi recebido, pago, está a vencer " +
				"e está atrasado. Use para perguntas sobre um mês específico, " +
				"inadimplência ou próximos vencimentos.",
			Parametros: esquema(map[string]any{
				"ano": map[string]any{"type": "integer", "description": "Ano. Padrão: o da tela."},
				"mes": map[string]any{"type": "integer", "description": "Mês de 1 a 12. Padrão: o da tela."},
			}, nil),
		},
		{
			Nome: "opcoes_de_filtro",
			Descricao: "Lista as empresas, contas correntes, departamentos e categorias " +
				"disponíveis. Use quando precisar saber o que existe antes de responder.",
			Parametros: esquema(map[string]any{}, nil),
		},
	}
}

func anoProp(descricao string) map[string]any {
	return map[string]any{"type": "integer", "description": descricao}
}

func esquema(props map[string]any, obrigatorios []string) map[string]any {
	if obrigatorios == nil {
		obrigatorios = []string{}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   obrigatorios,
	}
}

/*
Executor roda as ferramentas.

grupoID é campo da struct, e não parâmetro de ferramenta, de propósito: é o que
torna impossível a IA pedir dados de outro grupo. Não há argumento para isso
porque não existe o argumento.
*/
type Executor struct {
	pool    *pgxpool.Pool
	grupoID string
	ctxTela ContextoTela
	anon    *Anonimizador
}

func NovoExecutor(pool *pgxpool.Pool, grupoID string, ctxTela ContextoTela, anon *Anonimizador) *Executor {
	if ctxTela.Ano == 0 {
		ctxTela.Ano = time.Now().Year()
	}
	if ctxTela.Mes == 0 {
		ctxTela.Mes = int(time.Now().Month())
	}
	return &Executor{pool: pool, grupoID: grupoID, ctxTela: ctxTela, anon: anon}
}

// ErrFerramentaDesconhecida: o modelo pediu algo que não existe. Vira recusa,
// não pânico.
var ErrFerramentaDesconhecida = fmt.Errorf("ferramenta desconhecida")

/*
Executar roda a ferramenta e devolve o resultado JÁ PSEUDONIMIZADO, pronto para
ir ao provedor, mais a Fonte para a tela mostrar de onde veio o número.
*/
func (e *Executor) Executar(ctx context.Context, nome string, args json.RawMessage) (string, *Fonte, error) {
	fn, ok := e.despacho()[nome]
	if !ok {
		return "", nil, fmt.Errorf("%w: %q", ErrFerramentaDesconhecida, nome)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutFerramenta)
	defer cancel()

	return fn(ctx, e.parseArgs(args))
}

/*
despacho mapeia nome de ferramenta para implementação.

Tabela e não switch para que Conhece e Executar leiam a MESMA lista. Com duas
listas, uma ferramenta anunciada no catálogo e ausente do switch só apareceria
quando o modelo a chamasse em produção.
*/
func (e *Executor) despacho() map[string]func(context.Context, argsFerramenta) (string, *Fonte, error) {
	return map[string]func(context.Context, argsFerramenta) (string, *Fonte, error){
		"resumo_financeiro":              e.resumoFinanceiro,
		"abrir_por_categoria_ou_cliente": e.abrirPor,
		"fluxo_do_mes":                   e.fluxoDoMes,
		"opcoes_de_filtro":               func(ctx context.Context, _ argsFerramenta) (string, *Fonte, error) { return e.opcoesDeFiltro(ctx) },
	}
}

// Conhece diz se a ferramenta existe, sem executá-la.
func Conhece(nome string) bool {
	_, ok := (&Executor{}).despacho()[nome]
	return ok
}

/*
params monta os parâmetros da consulta.

É o ÚNICO lugar onde o grupo entra, e ele vem do campo do Executor — que foi
preenchido com as claims do token. A IA nunca toca aqui: não há argumento de
ferramenta que chegue a este campo.
*/
func (e *Executor) params(ano, mes int) dados.DashboardParams {
	return dados.DashboardParams{GrupoID: e.grupoID, Ano: ano, Mes: mes}
}

// argsFerramenta são os únicos parâmetros que a IA controla. Note a ausência de
// qualquer campo de grupo ou schema.
type argsFerramenta struct {
	Ano    int `json:"ano"`
	Mes    int `json:"mes"`
	AnoDe  int `json:"ano_de"`
	AnoAte int `json:"ano_ate"`
}

/*
parseArgs aplica os padrões da tela e recusa valores fora da faixa.

Argumento inválido do modelo cai no padrão em vez de virar erro: uma alucinação
de "mes": 13 não deve custar a resposta inteira ao usuário.
*/
func (e *Executor) parseArgs(raw json.RawMessage) argsFerramenta {
	p := argsFerramenta{Ano: e.ctxTela.Ano, Mes: e.ctxTela.Mes}
	if len(raw) == 0 {
		return p
	}

	var recebido argsFerramenta
	if err := json.Unmarshal(desembrulhar(raw), &recebido); err != nil {
		return p
	}
	if anoValido(recebido.Ano) {
		p.Ano = recebido.Ano
	}
	if recebido.Mes >= 1 && recebido.Mes <= 12 {
		p.Mes = recebido.Mes
	}
	if anoValido(recebido.AnoDe) {
		p.AnoDe = recebido.AnoDe
	}
	if anoValido(recebido.AnoAte) {
		p.AnoAte = recebido.AnoAte
	}
	return p
}

func anoValido(a int) bool { return a >= 2000 && a <= 2100 }

/*
desembrulhar trata o formato que os argumentos de ferramenta realmente têm.

No dialeto da OpenAI, `arguments` NÃO é um objeto JSON: é uma STRING contendo
JSON. O que chega aqui é `"{\"ano\":2025}"`, com aspas e escapes — e não
`{"ano":2025}`.

Isso passou despercebido porque a falha era silenciosa: o Unmarshal errava, o
parseArgs caía no padrão da tela, e a resposta vinha sobre o ano que o usuário
já estava olhando. Parecia certo em toda pergunta cujo ano fosse o da tela, que
é a maioria — mas "quanto foi em 2024?" vinha respondido sobre 2026, com
confiança e sem aviso.

Aceita as duas formas: alguns provedores compatíveis mandam o objeto direto.
*/
func desembrulhar(raw json.RawMessage) json.RawMessage {
	texto := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(texto, `"`) {
		return raw
	}
	var comoString string
	if err := json.Unmarshal([]byte(texto), &comoString); err != nil {
		return raw
	}
	return json.RawMessage(comoString)
}

/*
anos resolve o intervalo pedido numa lista concreta de anos.

Sem ano_de/ano_ate, devolve só o ano de sempre — o comportamento anterior, byte
por byte. Com intervalo, devolve os anos em ordem crescente, limitados a
maxAnosIntervalo.

O corte fica com os anos MAIS RECENTES, e não os primeiros: quem pede "dos
últimos cinco anos pra cá" quer chegar até hoje. O segundo retorno diz se houve
corte, porque um recorte silencioso numa comparação vira conclusão errada com
cara de certa.
*/
func (p argsFerramenta) anos() (lista []int, cortado bool) {
	de, ate := p.AnoDe, p.AnoAte
	if de == 0 && ate == 0 {
		return []int{p.Ano}, false
	}
	// Um lado só do intervalo: o outro é o ano de referência.
	if de == 0 {
		de = p.Ano
	}
	if ate == 0 {
		ate = p.Ano
	}
	if de > ate {
		de, ate = ate, de
	}
	if ate-de+1 > maxAnosIntervalo {
		de = ate - maxAnosIntervalo + 1
		cortado = true
	}
	for a := de; a <= ate; a++ {
		lista = append(lista, a)
	}
	return lista, cortado
}

func (e *Executor) resumoFinanceiro(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	anos, cortado := p.anos()

	porAno := make([]map[string]any, 0, len(anos))
	for _, ano := range anos {
		resp, err := dados.QueryDashboard(ctx, e.pool, e.params(ano, 0))
		if err != nil {
			return "", nil, fmt.Errorf("ia.tools.resumoFinanceiro(%d): %w", ano, err)
		}
		porAno = append(porAno, map[string]any{
			"ano":             ano,
			"receita_total":   resp.Cards.ReceitaTotal,
			"despesa_total":   resp.Cards.DespesaTotal,
			"resultado":       resp.Cards.Resultado,
			"saldo_em_contas": resp.Cards.SaldoContasCorrentes,
			"por_mes":         resp.GraficoMensal,
		})
	}

	// Um ano só mantém o formato de sempre — plano, sem um nível de aninhamento
	// que o modelo teria de atravessar à toa.
	var saida map[string]any
	if len(porAno) == 1 {
		saida = porAno[0]
	} else {
		saida = map[string]any{"anos": anos, "por_ano": porAno}
	}
	if cortado {
		saida["aviso_periodo"] = avisoCorte(anos)
	}

	return paraJSON(saida), &Fonte{
		Ferramenta: "resumo_financeiro",
		Filtros:    filtrosPeriodo(anos),
	}, nil
}

func avisoCorte(anos []int) string {
	return fmt.Sprintf(
		"O período pedido foi maior que o limite de %d anos e foi reduzido a %v. Diga isso ao usuário.",
		maxAnosIntervalo, anos)
}

func filtrosPeriodo(anos []int) map[string]any {
	if len(anos) == 1 {
		return map[string]any{"ano": anos[0]}
	}
	return map[string]any{"ano_de": anos[0], "ano_ate": anos[len(anos)-1]}
}

/*
PrepararRotulos carrega os nomes de clientes do grupo e os registra no
anonimizador ANTES de qualquer mensagem ser montada.

Resolve duas coisas de uma vez:

  - privacidade: o histórico é gravado com os nomes reais restaurados, e sem o
    mapa pronto ele voltaria em claro ao provedor a partir da segunda pergunta
    da conversa. Com os nomes já rotulados, a pseudonimização alcança o
    histórico também.
  - qualidade: sem isto, "Cliente A" significa um cliente diferente a cada
    pergunta, e o modelo lê o histórico com um vocabulário e os dados novos com
    outro.

Falha aqui não derruba a pergunta: perder a pré-população piora a conversa, mas
os resultados das ferramentas seguem pseudonimizados no caminho de sempre.
*/
func (e *Executor) PrepararRotulos(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, timeoutFerramenta)
	defer cancel()

	resp, err := dados.QueryFiltros(ctx, e.pool, e.params(e.ctxTela.Ano, 0))
	if err != nil {
		return fmt.Errorf("ia.tools.PrepararRotulos: %w", err)
	}

	// Ordem determinística: o rótulo de um cliente não pode depender da ordem em
	// que o banco resolveu devolver as linhas.
	nomes := append([]string(nil), resp.Clientes...)
	sort.Strings(nomes)
	if len(nomes) > maxNomesPreCarregados {
		nomes = nomes[:maxNomesPreCarregados]
	}
	for _, n := range nomes {
		e.anon.Rotular("Cliente", n)
	}

	empresas := make([]string, 0, len(resp.Empresas))
	for _, emp := range resp.Empresas {
		empresas = append(empresas, emp.Nome)
	}
	sort.Strings(empresas)
	for _, n := range empresas {
		e.anon.Rotular("Empresa", n)
	}
	return nil
}

func (e *Executor) abrirPor(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	anos, cortado := p.anos()

	if len(anos) == 1 {
		return e.abrirPorAno(ctx, anos[0], cortado, anos)
	}
	return e.abrirPorPeriodo(ctx, anos, cortado)
}

// abrirPorAno é o caminho de sempre: um ano, com os doze meses de cada linha.
func (e *Executor) abrirPorAno(ctx context.Context, ano int, cortado bool, anos []int) (string, *Fonte, error) {
	resp, err := dados.QueryPivot(ctx, e.pool, e.params(ano, 0))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.abrirPor: %w", err)
	}

	linhas := resp.Linhas
	truncado := false
	if len(linhas) > maxLinhasPivot {
		linhas = linhas[:maxLinhasPivot]
		truncado = true
	}

	// Os nomes de cliente saem daqui; categorias vão em claro porque são o
	// vocabulário que dá sentido à pergunta.
	simplificadas := make([]map[string]any, 0, len(linhas))
	for _, l := range linhas {
		simplificadas = append(simplificadas, map[string]any{
			"tipo":               l.Tipo,
			"categoria_superior": l.CategoriaSuperior,
			"categoria":          l.CategoriaFinal,
			"cliente":            e.anon.Rotular("Cliente", l.Cliente),
			"total":              l.Total,
			"meses":              l.Meses,
		})
	}

	saida := map[string]any{
		"ano":             ano,
		"linhas":          simplificadas,
		"resultado_total": resp.ResultadoTotal,
	}
	if truncado {
		saida["aviso"] = fmt.Sprintf(
			"Mostrando as %d primeiras de %d linhas. Diga ao usuário que o recorte foi limitado.",
			maxLinhasPivot, len(resp.Linhas))
	}
	if cortado {
		saida["aviso_periodo"] = avisoCorte(anos)
	}

	return paraJSON(saida), &Fonte{
		Ferramenta: "abrir_por_categoria_ou_cliente",
		Filtros:    map[string]any{"ano": ano},
	}, nil
}

// chavePivot identifica a MESMA linha entre anos diferentes. Sem os quatro
// campos, despesa e receita da mesma categoria somariam juntas.
type chavePivot struct {
	Tipo      string
	CatSuper  string
	Categoria string
	Cliente   string
}

/*
abrirPorPeriodo compara vários anos numa matriz única.

O ponto delicado está aqui, e não na consulta. Cada ano é truncado em
maxLinhasPivot pelo banco; se o recorte fosse aplicado por ano, o top de 2025 e
o de 2026 seriam conjuntos DIFERENTES, e um fornecedor grande num ano e ausente
do recorte no outro apareceria como se tivesse zerado. A comparação sairia
plausível e falsa — o pior tipo de erro num produto financeiro.

Por isso: une-se tudo, ordena-se pelo total do PERÍODO INTEIRO, e o recorte
acontece uma vez só. Ano sem movimento vira zero explícito, que é diferente de
"ficou de fora do recorte".
*/
func (e *Executor) abrirPorPeriodo(ctx context.Context, anos []int, cortado bool) (string, *Fonte, error) {
	totais := map[chavePivot]map[int]float64{}
	somaPeriodo := map[chavePivot]float64{}
	ordem := []chavePivot{}

	for _, ano := range anos {
		resp, err := dados.QueryPivot(ctx, e.pool, e.params(ano, 0))
		if err != nil {
			return "", nil, fmt.Errorf("ia.tools.abrirPor(%d): %w", ano, err)
		}
		for _, l := range resp.Linhas {
			k := chavePivot{l.Tipo, l.CategoriaSuperior, l.CategoriaFinal, l.Cliente}
			if _, visto := totais[k]; !visto {
				totais[k] = map[int]float64{}
				ordem = append(ordem, k)
			}
			totais[k][ano] += l.Total
			somaPeriodo[k] += math.Abs(l.Total)
		}
	}

	// Ordena pelo peso no período. O desempate por chave mantém a saída estável
	// entre execuções — um mapa em Go não tem ordem.
	sort.SliceStable(ordem, func(i, j int) bool {
		a, b := ordem[i], ordem[j]
		if somaPeriodo[a] != somaPeriodo[b] {
			return somaPeriodo[a] > somaPeriodo[b]
		}
		return chaveTexto(a) < chaveTexto(b)
	})

	totalLinhas := len(ordem)
	if len(ordem) > maxLinhasPivot {
		ordem = ordem[:maxLinhasPivot]
	}

	linhas := make([]map[string]any, 0, len(ordem))
	for _, k := range ordem {
		porAno := make(map[string]float64, len(anos))
		for _, ano := range anos {
			// Zero explícito: a ausência precisa ser legível como "não houve
			// movimento", e não como buraco no dado.
			porAno[strconv.Itoa(ano)] = totais[k][ano]
		}
		linhas = append(linhas, map[string]any{
			"tipo":               k.Tipo,
			"categoria_superior": k.CatSuper,
			"categoria":          k.Categoria,
			"cliente":            e.anon.Rotular("Cliente", k.Cliente),
			"por_ano":            porAno,
			"total_periodo":      somaPeriodo[k],
		})
	}

	saida := map[string]any{
		"anos":   anos,
		"linhas": linhas,
		"nota": "Cada linha traz o total de CADA ano em por_ano. Zero significa " +
			"que não houve movimento naquele ano, não que o dado falta.",
	}
	if totalLinhas > len(ordem) {
		saida["aviso"] = fmt.Sprintf(
			"Mostrando as %d maiores de %d combinações, ordenadas pelo total do período inteiro. "+
				"Diga ao usuário que o recorte foi limitado.", len(ordem), totalLinhas)
	}
	if cortado {
		saida["aviso_periodo"] = avisoCorte(anos)
	}

	return paraJSON(saida), &Fonte{
		Ferramenta: "abrir_por_categoria_ou_cliente",
		Filtros:    filtrosPeriodo(anos),
	}, nil
}

func chaveTexto(k chavePivot) string {
	return k.Tipo + "|" + k.CatSuper + "|" + k.Categoria + "|" + k.Cliente
}

func (e *Executor) fluxoDoMes(ctx context.Context, p argsFerramenta) (string, *Fonte, error) {
	resp, err := dados.QueryFluxoCaixa(ctx, e.pool, e.params(p.Ano, p.Mes))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.fluxoDoMes: %w", err)
	}

	// O resumo e os próximos vencimentos bastam para quase toda pergunta; a
	// lista inteira de lançamentos seriam milhares de linhas de token pago.
	vencimentos := make([]map[string]any, 0, len(resp.ProximosVencimentos))
	for _, v := range resp.ProximosVencimentos {
		vencimentos = append(vencimentos, map[string]any{
			"data":      v.Data,
			"descricao": e.anon.Rotular("Cliente", v.Descricao),
			"categoria": v.Categoria,
			"tipo":      v.Tipo,
			"valor":     v.Valor,
			"status":    v.Status,
		})
	}

	saida := map[string]any{
		"ano":                  p.Ano,
		"mes":                  p.Mes,
		"resumo":               resp.Resumo,
		"total_lancamentos":    len(resp.Transacoes),
		"proximos_vencimentos": vencimentos,
	}
	return paraJSON(saida), &Fonte{
		Ferramenta: "fluxo_do_mes",
		Filtros:    map[string]any{"ano": p.Ano, "mes": p.Mes},
	}, nil
}

func (e *Executor) opcoesDeFiltro(ctx context.Context) (string, *Fonte, error) {
	resp, err := dados.QueryFiltros(ctx, e.pool, e.params(e.ctxTela.Ano, 0))
	if err != nil {
		return "", nil, fmt.Errorf("ia.tools.opcoesDeFiltro: %w", err)
	}

	empresas := make([]string, 0, len(resp.Empresas))
	for _, emp := range resp.Empresas {
		empresas = append(empresas, e.anon.Rotular("Empresa", emp.Nome))
	}

	saida := map[string]any{
		"empresas":      empresas,
		"categorias":    resp.Categorias,
		"departamentos": resp.Departamentos,
	}
	return paraJSON(saida), &Fonte{
		Ferramenta: "opcoes_de_filtro",
		Filtros:    map[string]any{},
	}, nil
}

// paraJSON serializa para o modelo. Falha de serialização vira um objeto de
// erro legível em vez de string vazia — o modelo sabe dizer "não consegui".
func paraJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"erro":"não foi possível serializar o resultado da consulta"}`
	}
	return string(b)
}
