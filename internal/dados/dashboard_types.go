package dados

type DashboardResponse struct {
	Cards                     CardMetrics        `json:"cards"`
	GraficoMensal             []GraficoMensal    `json:"grafico_mensal"`
	GraficoResultadoAcumulado []GraficoAcumulado `json:"grafico_resultado_acumulado"`
	// SaldosMensais traz os doze saldos de uma vez para que a tela possa recortar
	// meses sem uma ida ao servidor por clique. Saldo é posição num instante, não
	// soma de período: recortar no cliente exige ter cada instante pronto.
	SaldosMensais      []SaldoMes         `json:"saldos_mensais"`
	FiltrosDisponiveis FiltrosDisponiveis `json:"filtros_disponiveis"`

	/*
	 * AncoraAplicavel diz se o acumulado parte do saldo das contas ou do zero.
	 *
	 * Falso quando há filtro que o saldo não sabe responder — categoria,
	 * departamento, cliente. A série continua correta; o que muda é a pergunta
	 * que ela responde, e a tela precisa saber disso para trocar o título em vez
	 * de apresentar um acumulado do recorte como se fosse posição de caixa.
	 */
	AncoraAplicavel bool `json:"ancora_aplicavel"`
	// MesCorte é a fronteira realizado/previsto, no mesmo critério da aba
	// Resultado: mês corrente no ano corrente, 13 num ano passado, 1 num futuro.
	MesCorte int `json:"mes_corte"`
}

// SaldoMes é o saldo das contas correntes ao FIM do mês: o saldo inicial
// cadastrado mais os movimentos já realizados até ali.
//
// Só conta movimento realizado (mov_ou_extrato = 'mov'). Somar as provisões do
// extrato daria saldo previsto, que é outra coisa — e faria o saldo de dezembro
// aparecer como dinheiro em caixa hoje.
type SaldoMes struct {
	Mes     int     `json:"mes"`
	MesNome string  `json:"mes_nome"`
	Saldo   float64 `json:"saldo"`
}

type CardMetrics struct {
	ReceitaTotal         float64 `json:"receita_total"`
	DespesaTotal         float64 `json:"despesa_total"`
	Resultado            float64 `json:"resultado"`
	SaldoContasCorrentes float64 `json:"saldo_contas_correntes"`
	// SaldoInicial é a soma de contas_correntes.saldo_inicial, antes de qualquer
	// movimento — a abertura da cascata. Explícito na resposta porque derivá-lo
	// no cliente exigiria subtrair o realizado de janeiro do saldo de janeiro, e
	// o cliente só tem a série que mistura realizado e previsto: a conta daria
	// um número plausível e errado.
	SaldoInicial float64 `json:"saldo_inicial"`
}

type GraficoMensal struct {
	Mes     int     `json:"mes"`
	MesNome string  `json:"mes_nome"`
	Receita float64 `json:"receita"`
	Despesa float64 `json:"despesa"`
	// ResultadoMes soma as duas trilhas. Mantido como estava: é o que a tela
	// desenha hoje, e mudá-lo alteraria números sem ninguém pedir.
	ResultadoMes float64 `json:"resultado_mes"`

	/*
	 * As duas trilhas do modelo do Omie.
	 *
	 * REALIZADO é dinheiro que já entrou ou saiu da conta (mov_ou_extrato =
	 * 'mov'); PREVISTO é como o mês deve fechar — o realizado dele MAIS as
	 * provisões ainda por acontecer.
	 *
	 * Separar importa porque a tela hoje mistura os dois sem avisar: o card
	 * "Saldo em contas" soma só o realizado e a cascata ao lado inclui as
	 * provisões, e os dois se chamam saldo.
	 */
	ReceitaRealizada   float64 `json:"receita_realizada"`
	DespesaRealizada   float64 `json:"despesa_realizada"`
	ResultadoRealizado float64 `json:"resultado_realizado"`
	ReceitaPrevista    float64 `json:"receita_prevista"`
	DespesaPrevista    float64 `json:"despesa_prevista"`
	ResultadoPrevisto  float64 `json:"resultado_previsto"`
}

type GraficoAcumulado struct {
	Mes          int     `json:"mes"`
	MesNome      string  `json:"mes_nome"`
	ResultadoMes float64 `json:"resultado_mes"`
	Acumulado    float64 `json:"acumulado"`

	/*
	 * O acumulado nas duas trilhas, cada uma partindo da MESMA âncora e
	 * encadeando mês a mês — a conta do Omie:
	 *
	 *     SaldoInicial(N) = Acumulado(N-1)
	 *     Acumulado(N)    = SaldoInicial(N) + Resultado(N)
	 *
	 * SaldoInicial vem junto porque é o que a aba Fluxo de Caixa mostra como
	 * abertura do mês, e derivá-lo na tela exigiria olhar o mês anterior.
	 */
	SaldoInicialRealizado float64 `json:"saldo_inicial_realizado"`
	ResultadoRealizado    float64 `json:"resultado_realizado"`
	AcumuladoRealizado    float64 `json:"acumulado_realizado"`
	SaldoInicialPrevisto  float64 `json:"saldo_inicial_previsto"`
	ResultadoPrevisto     float64 `json:"resultado_previsto"`
	AcumuladoPrevisto     float64 `json:"acumulado_previsto"`
}

type FiltrosDisponiveis struct {
	ContasCorrentes []ContaCorrenteItem `json:"contas_correntes"`
	Departamentos   []string            `json:"departamentos"`
	Categorias      []string            `json:"categorias"`
	Clientes        []string            `json:"clientes"`
	Empresas        []EmpresaItem       `json:"empresas"`
}

type ContaCorrenteItem struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
	// FluxoCaixa é a marca cFluxoCaixa do Omie, usada só para agrupar o filtro:
	// 'S' = considerada no fluxo de caixa, 'N' = não considerada. Vazio = conta
	// que o extrato ainda não sincronizou, e por isso não se sabe.
	FluxoCaixa string `json:"fluxo_caixa"`
}

type EmpresaItem struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

var nomeMes = [13]string{
	"", "Jan", "Fev", "Mar", "Abr", "Mai", "Jun",
	"Jul", "Ago", "Set", "Out", "Nov", "Dez",
}
