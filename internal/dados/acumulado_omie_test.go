package dados

import (
	"math"
	"testing"
)

/*
A conta do Omie, conferida contra um relatório real.

Estes números saíram de uma extração do relatório de Fluxo de Caixa do Omie —
exercício inteiro, 13.806 linhas — e existem aqui por um motivo: sem eles, o
acumulado só poderia ser conferido contra ele mesmo. Um erro de modelo (acumular
o que não devia, ancorar no lugar errado, somar o previsto duas vezes) produz uma
série internamente consistente e completamente errada, e nenhum teste de
propriedade pega isso. Estes pegam, porque a resposta certa veio de fora.

O relatório original encadeava o saldo inicial de um mês a partir do acumulado
do anterior (Saldo Inicial(N) = Acumulado(N−1)). A nossa implementação NÃO
encadeia mais: cada mês recebe seu próprio saldo real (vindo de
saldo_por_contas, não mais de um saldo de cadastro sem data), e por isso os
testes abaixo alimentam `acumularSaldos`/`acumularResultado` com a série de
SaldoIniRealizado do relatório inteira, mês a mês, em vez de só a âncora de
janeiro — o relatório real já tem essa série (ela é, sob o modelo antigo,
idêntica ao acumulado do mês anterior; sob o modelo novo, cada valor seria lido
direto de saldo_por_contas). O que os testes continuam conferindo é a
aritmética Resultado(N) + Saldo Inicial(N) = Acumulado(N), não mais o
encadeamento entre meses.

A regra que menos se adivinha, e que continua valendo: a trilha PREVISTA
acumula SOBRE a realizada — Resultado Previsto = Receitas Previstas + Despesas
Previstas + Resultado REALIZADO do mesmo mês. A projeção do mês inclui o que já
aconteceu nele.

Os valores são de um cliente real e estão aqui SEM qualquer identificação — nem
empresa, nem conta, nem fornecedor. O que importa para o teste é a aritmética.
*/

type mesOmie struct {
	Mes                int
	SaldoIniRealizado  float64
	ReceitaRealizada   float64
	DespesaRealizada   float64
	ResultadoRealizado float64
	AcumuladoRealizado float64
	ReceitaPrevista    float64
	DespesaPrevista    float64
	ResultadoPrevisto  float64
}

var relatorioOmie = []mesOmie{
	{Mes: 1, SaldoIniRealizado: 204780.39, ReceitaRealizada: 3108305.05, DespesaRealizada: -3188860.53, ResultadoRealizado: -80555.48, AcumuladoRealizado: 124224.91, ReceitaPrevista: 11609.35, DespesaPrevista: -130612.12, ResultadoPrevisto: -199558.25},
	{Mes: 2, SaldoIniRealizado: 124224.91, ReceitaRealizada: 3377216.01, DespesaRealizada: -3322149.58, ResultadoRealizado: 55066.43, AcumuladoRealizado: 179291.34, ReceitaPrevista: 4649.99, DespesaPrevista: -86990.23, ResultadoPrevisto: -27273.81},
	{Mes: 3, SaldoIniRealizado: 179291.34, ReceitaRealizada: 3332758.81, DespesaRealizada: -3273191.64, ResultadoRealizado: 59567.17, AcumuladoRealizado: 238858.51, ReceitaPrevista: 28385.09, DespesaPrevista: -193975.79, ResultadoPrevisto: -106023.53},
	{Mes: 4, SaldoIniRealizado: 238858.51, ReceitaRealizada: 2729902.90, DespesaRealizada: -2680879.47, ResultadoRealizado: 49023.43, AcumuladoRealizado: 287881.94, ReceitaPrevista: 1156.48, DespesaPrevista: -84708.85, ResultadoPrevisto: -34528.94},
	{Mes: 5, SaldoIniRealizado: 287881.94, ReceitaRealizada: 3116613.36, DespesaRealizada: -2587506.94, ResultadoRealizado: 529106.42, AcumuladoRealizado: 816988.36, ReceitaPrevista: 33058.49, DespesaPrevista: -141450.85, ResultadoPrevisto: 420714.06},
	{Mes: 6, SaldoIniRealizado: 816988.36, ReceitaRealizada: 3053969.44, DespesaRealizada: -3665127.41, ResultadoRealizado: -611157.97, AcumuladoRealizado: 205830.39, ReceitaPrevista: 207030.28, DespesaPrevista: -286966.55, ResultadoPrevisto: -691094.24},
	{Mes: 7, SaldoIniRealizado: 205830.39, ReceitaRealizada: 4595556.97, DespesaRealizada: -3684355.97, ResultadoRealizado: 911201.00, AcumuladoRealizado: 1117031.39, ReceitaPrevista: 202343.05, DespesaPrevista: -134433.04, ResultadoPrevisto: 979111.01},
	{Mes: 8, SaldoIniRealizado: 1117031.39, ReceitaRealizada: 2730372.90, DespesaRealizada: -3535827.00, ResultadoRealizado: -805454.10, AcumuladoRealizado: 311577.29, ReceitaPrevista: 154688.82, DespesaPrevista: -501469.30, ResultadoPrevisto: -1152234.58},
	{Mes: 9, SaldoIniRealizado: 311577.29, ReceitaRealizada: 3325393.99, DespesaRealizada: -3331918.14, ResultadoRealizado: -6524.15, AcumuladoRealizado: 305053.14, ReceitaPrevista: 357788.23, DespesaPrevista: -630277.51, ResultadoPrevisto: -279013.43},
	// Outubro é o mês corrente da extração: tem realizado E previsto, e é onde a
	// fronteira das duas trilhas aparece.
	{Mes: 10, SaldoIniRealizado: 305053.14, ReceitaRealizada: 848172.94, DespesaRealizada: -980954.53, ResultadoRealizado: -132781.59, AcumuladoRealizado: 172271.55, ReceitaPrevista: 3557011.36, DespesaPrevista: -2313076.06, ResultadoPrevisto: 1111153.71},
	// Novembro e dezembro não têm realizado nenhum — o acumulado realizado
	// simplesmente não se move, e é por isso que ele repete 172271.55.
	{Mes: 11, SaldoIniRealizado: 172271.55, ResultadoRealizado: 0, AcumuladoRealizado: 172271.55, ReceitaPrevista: 2288304.87, DespesaPrevista: -2999399.35, ResultadoPrevisto: -711094.48},
	{Mes: 12, SaldoIniRealizado: 172271.55, ResultadoRealizado: 0, AcumuladoRealizado: 172271.55, ReceitaPrevista: 557001.41, DespesaPrevista: -1281566.40, ResultadoPrevisto: -724564.99},
}

// A tolerância é de um centavo: o relatório vem com duas casas e a soma de
// milhares de linhas carrega arredondamento. Mais que isso seria erro de conta.
const centavo = 0.011

func perto(a, b float64) bool { return math.Abs(a-b) < centavo }

// ancoraMensal repete um único valor nas 12 posições — usado nos testes que
// não testam variação mensal da âncora, só a aritmética de um mês isolado.
func ancoraMensal(v float64) [12]float64 {
	var out [12]float64
	for i := range out {
		out[i] = v
	}
	return out
}

func TestOmie_ResultadoEhReceitaMaisDespesa(t *testing.T) {
	for _, m := range relatorioOmie {
		if soma := m.ReceitaRealizada + m.DespesaRealizada; !perto(soma, m.ResultadoRealizado) {
			t.Errorf("mês %d: receita+despesa = %.2f, mas o resultado é %.2f", m.Mes, soma, m.ResultadoRealizado)
		}
	}
}

func TestOmie_AcumuladoEhAncoraMaisResultado(t *testing.T) {
	for _, m := range relatorioOmie {
		if soma := m.SaldoIniRealizado + m.ResultadoRealizado; !perto(soma, m.AcumuladoRealizado) {
			t.Errorf("mês %d: saldo+resultado = %.2f, mas o acumulado é %.2f", m.Mes, soma, m.AcumuladoRealizado)
		}
	}
}

/*
A regra menos óbvia, e a mais fácil de implementar errado.

A trilha prevista NÃO é "só o que falta acontecer": ela inclui o realizado do
próprio mês. Implementar como soma isolada das previsões daria um número menor e
plausível, e a projeção de caixa ficaria otimista justamente nos meses em que já
houve movimento.
*/
func TestOmie_PrevistoAcumulaSobreORealizado(t *testing.T) {
	for _, m := range relatorioOmie {
		soma := m.ReceitaPrevista + m.DespesaPrevista + m.ResultadoRealizado
		if !perto(soma, m.ResultadoPrevisto) {
			t.Errorf("mês %d: previsto+realizado = %.2f, mas o resultado previsto é %.2f",
				m.Mes, soma, m.ResultadoPrevisto)
		}
	}
}

/*
Agora a nossa implementação contra os mesmos números.

acumularSaldos já existe e faz a acumulação da trilha realizada. Alimentada com
a âncora e os resultados mensais do relatório, ela tem de reproduzir a coluna de
acumulado do Omie — mês a mês, não só no total.
*/
func TestAcumularSaldos_ReproduzOAcumuladoDoOmie(t *testing.T) {
	var realizado, saldoMensal [12]float64
	for i, m := range relatorioOmie {
		realizado[i] = m.ResultadoRealizado
		saldoMensal[i] = m.SaldoIniRealizado
	}

	saldos := acumularSaldos(saldoMensal, realizado)

	if len(saldos) != 12 {
		t.Fatalf("devolveu %d meses", len(saldos))
	}
	for i, s := range saldos {
		esperado := relatorioOmie[i].AcumuladoRealizado
		if !perto(s.Saldo, esperado) {
			t.Errorf("mês %d: acumulado %.2f, o Omie diz %.2f", s.Mes, s.Saldo, esperado)
		}
	}
}

// Mês sem movimento não muda o acumulado — é o caso de novembro e dezembro na
// extração, e o que garante que um mês vazio não zera a série.
func TestAcumularSaldos_MesSemMovimentoMantemOAcumulado(t *testing.T) {
	saldos := acumularSaldos(ancoraMensal(1000), [12]float64{})

	for _, s := range saldos {
		if !perto(s.Saldo, 1000) {
			t.Fatalf("mês %d mexeu no acumulado sem movimento: %.2f", s.Mes, s.Saldo)
		}
	}
}

/*
A implementação contra o relatório, mês a mês.

Este é o teste que decide se o recurso presta: alimentado com as receitas e
despesas do relatório, acumularResultado tem de devolver exatamente as colunas
"Saldo Inicial", "Resultado" e "Resultado Acumulado" que o Omie imprime — nas
duas trilhas.
*/
func TestAcumularResultado_ReproduzORelatorioDoOmie(t *testing.T) {
	mensal := make([]GraficoMensal, 0, len(relatorioOmie))
	var saldoMensal [12]float64
	for i, m := range relatorioOmie {
		saldoMensal[i] = m.SaldoIniRealizado
		mensal = append(mensal, GraficoMensal{
			Mes:                m.Mes,
			MesNome:            nomeMes[m.Mes],
			ReceitaRealizada:   m.ReceitaRealizada,
			DespesaRealizada:   -m.DespesaRealizada, // a view guarda despesa positiva
			ResultadoRealizado: m.ResultadoRealizado,
			ReceitaPrevista:    m.ReceitaPrevista,
			DespesaPrevista:    -m.DespesaPrevista,
			ResultadoPrevisto:  m.ResultadoPrevisto,
		})
	}

	out := acumularResultado(saldoMensal, 0, mensal)

	for i, got := range out {
		esperado := relatorioOmie[i]

		if !perto(got.SaldoInicialRealizado, esperado.SaldoIniRealizado) {
			t.Errorf("mês %d: saldo inicial %.2f, o Omie diz %.2f",
				got.Mes, got.SaldoInicialRealizado, esperado.SaldoIniRealizado)
		}
		if !perto(got.AcumuladoRealizado, esperado.AcumuladoRealizado) {
			t.Errorf("mês %d: acumulado realizado %.2f, o Omie diz %.2f",
				got.Mes, got.AcumuladoRealizado, esperado.AcumuladoRealizado)
		}
	}
}

/*
A independência entre meses, verificada na saída da nossa função.

É a propriedade central do modelo novo: um mês ruim não pode contaminar os
seguintes. Alterando o saldo real de um único mês, só o acumulado DAQUELE mês
deve mudar — os outros, que partem do próprio saldo real deles, têm que
continuar exatamente iguais.
*/
func TestAcumularResultado_MesesSaoIndependentes(t *testing.T) {
	mensal := []GraficoMensal{
		{Mes: 1, ResultadoRealizado: 100, ResultadoPrevisto: 150},
		{Mes: 2, ResultadoRealizado: -40, ResultadoPrevisto: -10},
		{Mes: 3, ResultadoRealizado: 25, ResultadoPrevisto: 25},
	}

	base := ancoraMensal(1000)
	alterado := base
	alterado[0] = 1000000 // um saldo real de janeiro muito diferente

	out := acumularResultado(base, 0, mensal)
	outAlterado := acumularResultado(alterado, 0, mensal)

	if perto(out[0].AcumuladoRealizado, outAlterado[0].AcumuladoRealizado) {
		t.Fatalf("mês 1 não reagiu à mudança no seu próprio saldo real")
	}
	for i := 1; i < len(out); i++ {
		if !perto(out[i].AcumuladoRealizado, outAlterado[i].AcumuladoRealizado) {
			t.Errorf("mês %d mudou (%.2f -> %.2f) por causa de um saldo alterado em outro mês — encadeamento não deveria mais existir",
				out[i].Mes, out[i].AcumuladoRealizado, outAlterado[i].AcumuladoRealizado)
		}
		if !perto(out[i].AcumuladoPrevisto, outAlterado[i].AcumuladoPrevisto) {
			t.Errorf("mês %d (previsto) mudou (%.2f -> %.2f) por causa de um saldo alterado em outro mês",
				out[i].Mes, out[i].AcumuladoPrevisto, outAlterado[i].AcumuladoPrevisto)
		}
	}
}

// As duas trilhas partem do mesmo ponto: a posição de caixa no início do período
// é uma só. Elas divergem a partir do primeiro mês, não antes dele.
func TestAcumularResultado_AsDuasTrilhasAbremNaMesmaAncora(t *testing.T) {
	out := acumularResultado(ancoraMensal(500), 0, []GraficoMensal{{Mes: 1, ResultadoRealizado: 10, ResultadoPrevisto: 99}})

	if !perto(out[0].SaldoInicialRealizado, 500) || !perto(out[0].SaldoInicialPrevisto, 500) {
		t.Fatalf("abriram em pontos diferentes: real=%.2f prev=%.2f",
			out[0].SaldoInicialRealizado, out[0].SaldoInicialPrevisto)
	}
	if perto(out[0].AcumuladoRealizado, out[0].AcumuladoPrevisto) {
		t.Fatal("as trilhas não divergiram, apesar de resultados diferentes")
	}
}

/*
Âncora zero é um caso de uso, não um defeito.

É o que acontece quando há filtro de categoria: a série vira acumulado do
recorte, a partir do zero. Precisa continuar somando normalmente.
*/
func TestAcumularResultado_AncoraZeroAcumulaDoZero(t *testing.T) {
	out := acumularResultado([12]float64{}, 0, []GraficoMensal{
		{Mes: 1, ResultadoRealizado: 100},
		{Mes: 2, ResultadoRealizado: 100},
	})

	if !perto(out[0].AcumuladoRealizado, 100) || !perto(out[1].AcumuladoRealizado, 200) {
		t.Fatalf("acumulado = %.2f, %.2f", out[0].AcumuladoRealizado, out[1].AcumuladoRealizado)
	}
}

/*
A âncora do saldo só vale para os filtros que o saldo sabe responder.

Categoria, departamento e cliente são dimensões do lançamento; o extrato
bancário não sabe quanto do saldo veio de "Despesas com Pessoal". Ancorar
mesmo assim somaria o movimento de um recorte a uma abertura do total.
*/
func TestAncoraAplicavel(t *testing.T) {
	casos := []struct {
		nome     string
		params   DashboardParams
		esperado bool
	}{
		{"sem filtro nenhum", DashboardParams{Ano: 2026}, true},
		{"empresa o saldo sabe", DashboardParams{Empresas: []string{"a"}}, true},
		{"conta corrente o saldo sabe", DashboardParams{ContasCorrentes: []string{"1"}}, true},
		{"categoria não", DashboardParams{Categorias: []string{"Folha"}}, false},
		{"departamento não", DashboardParams{Departamentos: []string{"Vendas"}}, false},
		{"cliente não", DashboardParams{Cliente: "Alpha"}, false},
		/*
		 * Excluir NAO e recortar.
		 *
		 * A tela exclui transferencias por padrao. Tratar isso como recorte faria
		 * o acumulado nunca partir do saldo na configuracao normal — o recurso
		 * seria inutil justamente no estado em que todo mundo abre a tela.
		 */
		{"exclusão de categoria é limpeza, não recorte", DashboardParams{CategoriasExcluir: []string{"Transf."}}, true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := ancoraAplicavel(c.params); got != c.esperado {
				t.Fatalf("ancoraAplicavel = %v, esperava %v", got, c.esperado)
			}
		})
	}
}

/*
Atrasados pesam na projeção, nunca no realizado.

É a diferença entre "quanto tenho" e "quanto vou ter": um título vencido e não
pago ainda não mexeu na conta — então não pode aparecer no acumulado realizado —
mas vai mexer, e por isso entra na abertura da trilha prevista. É o que o
relatório do Omie faz com o balde (Atrasados), antes de janeiro.
*/
func TestAcumularResultado_AtrasadosSoNaTrilhaPrevista(t *testing.T) {
	mensal := []GraficoMensal{{Mes: 1, ResultadoRealizado: 100, ResultadoPrevisto: 100}}

	semAtraso := acumularResultado(ancoraMensal(1000), 0, mensal)
	comAtraso := acumularResultado(ancoraMensal(1000), -300, mensal)

	if !perto(semAtraso[0].AcumuladoRealizado, comAtraso[0].AcumuladoRealizado) {
		t.Errorf("os atrasados mexeram no realizado: %.2f virou %.2f",
			semAtraso[0].AcumuladoRealizado, comAtraso[0].AcumuladoRealizado)
	}
	if !perto(comAtraso[0].SaldoInicialPrevisto, 700) {
		t.Errorf("abertura prevista = %.2f, esperava 1000−300", comAtraso[0].SaldoInicialPrevisto)
	}
	if !perto(comAtraso[0].AcumuladoPrevisto, 800) {
		t.Errorf("acumulado previsto = %.2f, esperava 700+100", comAtraso[0].AcumuladoPrevisto)
	}
}

// Com o filtro desligado (atrasados = 0) as duas trilhas abrem no mesmo ponto —
// é o comportamento anterior, preservado.
func TestAcumularResultado_SemAtrasadosAsTrilhasAbremJuntas(t *testing.T) {
	out := acumularResultado(ancoraMensal(1000), 0, []GraficoMensal{{Mes: 1}})

	if !perto(out[0].SaldoInicialRealizado, out[0].SaldoInicialPrevisto) {
		t.Fatalf("abriram separadas sem atrasados: %.2f e %.2f",
			out[0].SaldoInicialRealizado, out[0].SaldoInicialPrevisto)
	}
}
