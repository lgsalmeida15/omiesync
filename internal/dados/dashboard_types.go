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
	Mes          int     `json:"mes"`
	MesNome      string  `json:"mes_nome"`
	Receita      float64 `json:"receita"`
	Despesa      float64 `json:"despesa"`
	ResultadoMes float64 `json:"resultado_mes"`
}

type GraficoAcumulado struct {
	Mes          int     `json:"mes"`
	MesNome      string  `json:"mes_nome"`
	ResultadoMes float64 `json:"resultado_mes"`
	Acumulado    float64 `json:"acumulado"`
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
