package dados

type DashboardResponse struct {
	Cards                    CardMetrics          `json:"cards"`
	GraficoMensal            []GraficoMensal      `json:"grafico_mensal"`
	GraficoResultadoAcumulado []GraficoAcumulado  `json:"grafico_resultado_acumulado"`
	FiltrosDisponiveis       FiltrosDisponiveis   `json:"filtros_disponiveis"`
}

type CardMetrics struct {
	ReceitaTotal          float64 `json:"receita_total"`
	DespesaTotal          float64 `json:"despesa_total"`
	Resultado             float64 `json:"resultado"`
	SaldoContasCorrentes  float64 `json:"saldo_contas_correntes"`
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
	// FluxoCaixa é a marca cFluxoCaixa do Omie, usada só para agrupar o filtro.
	// Atenção à semântica, que é invertida em relação ao nome: 'S' = NÃO
	// considerada no fluxo de caixa, 'N' = considerada. Vazio = conta que o
	// extrato ainda não sincronizou, e por isso não se sabe.
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
