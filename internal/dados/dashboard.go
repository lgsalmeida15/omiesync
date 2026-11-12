package dados

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ErrViewNaoPopulada indica que a view materializada existe mas nunca recebeu
// REFRESH. Acontece logo após um re-provisionamento, que recria as views WITH NO
// DATA — a histórica só é repopulada no sync seguinte. O Postgres trata leitura
// nessa situação como erro (SQLSTATE 55000), não como resultado vazio.
var ErrViewNaoPopulada = errors.New("dados ainda não disponíveis: aguarde a conclusão do próximo sync")

// isViewNaoPopulada identifica o SQLSTATE 55000 (object_not_in_prerequisite_state).
func isViewNaoPopulada(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55000"
}

type DashboardParams struct {
	GrupoID         string
	Ano             int
	Empresas        []string
	ContasCorrentes []string
	Departamentos   []string
	Categorias      []string
	Cliente         string

	// Inadimplencia inclui os titulos vencidos e nao pagos (status ATRASADO de
	// contas_pagar/contas_receber) no fluxo. Fica desligado por padrao no
	// backend: sem o parametro a resposta e identica a de antes desta opcao
	// existir, e nenhum consumidor herda o comportamento novo sem pedir.
	Inadimplencia bool

	// Mes (1-12) só é usado pelo fluxo de caixa. Fica fora de buildFiltro de
	// propósito: dashboard e pivot são anuais, e filtrar por mês ali esvaziaria
	// onze das doze colunas.
	Mes int

	// CategoriasExcluir remove categorias do resultado. É lista de exclusão, e não
	// de inclusão, de propósito: com inclusão, uma categoria nova no Omie ficaria
	// fora dos números em silêncio até alguém marcá-la. Excluindo, o que é novo
	// entra automaticamente.
	//
	// Não afeta as listas de opções — a categoria excluída precisa continuar
	// aparecendo no filtro para que o usuário possa incluí-la de volta.
	CategoriasExcluir []string

	/*
	 * Trilhas recorta por natureza do lançamento: "realizado" é dinheiro que já
	 * entrou ou saiu da conta (mov_ou_extrato = 'mov'); "previsto" é provisão do
	 * extrato, o que ainda vai acontecer ('ext').
	 *
	 * Lista vazia = as duas, pela mesma convenção dos outros filtros marcáveis
	 * do produto. Isso mantém a resposta idêntica à de antes para quem não pedir
	 * nada — nenhum consumidor herda recorte sem escolher.
	 *
	 * Fica em buildFiltro, e não num tratamento por consulta, porque é o que faz
	 * o dashboard, a aba Resultado e o Fluxo de Caixa recortarem juntos: as três
	 * leem a mesma base, e um filtro aplicado em só uma delas faria as abas
	 * discordarem sobre o mesmo mês.
	 */
	Trilhas []string
}

// Valores aceitos em Trilhas. Qualquer outro é ignorado no parse.
const (
	TrilhaRealizado = "realizado"
	TrilhaPrevisto  = "previsto"
)

// trilhasParaFonte traduz a escolha do usuário para os valores da coluna
// mov_ou_extrato. Devolve nil quando não há recorte a fazer — nenhuma trilha
// marcada, ou as duas, que dão no mesmo.
func trilhasParaFonte(trilhas []string) []string {
	var fontes []string
	for _, t := range trilhas {
		switch t {
		case TrilhaRealizado:
			fontes = append(fontes, "mov")
		case TrilhaPrevisto:
			fontes = append(fontes, "ext")
		}
	}
	if len(fontes) == 0 || len(fontes) == 2 {
		return nil
	}
	return fontes
}

// nivelFiltro identifica até onde aplicar os filtros. As opções de cada filtro são
// restringidas apenas pelos níveis ACIMA dele: se as categorias disponíveis fossem
// filtradas pela categoria selecionada, marcar uma faria as outras desaparecerem da
// lista e o multi-select deixaria de funcionar.
type nivelFiltro int

const (
	nivelAno nivelFiltro = iota
	nivelEmpresas
	nivelContas
	nivelDepartamentos
	nivelCategorias
	nivelTodos
)

// schemaForGrupo resolve o schema_name do grupo.
func schemaForGrupo(ctx context.Context, pool *pgxpool.Pool, grupoID string) (string, error) {
	var schema string
	err := pool.QueryRow(ctx,
		`SELECT schema_name FROM _etl.grupos WHERE id = $1 AND deleted_at IS NULL`,
		grupoID,
	).Scan(&schema)
	if err != nil {
		return "", fmt.Errorf("dados.schemaForGrupo: %w", err)
	}
	return schema, nil
}

// viewName escolhe a view correta baseado no ano solicitado vs ano atual.
func viewName(ano int) string {
	if ano >= time.Now().Year() {
		return "matvw_gerencial_ano_corrente"
	}
	return "matvw_gerencial_historico"
}

// QueryDashboard executa todas as queries necessárias e monta o DashboardResponse.
func QueryDashboard(ctx context.Context, pool *pgxpool.Pool, p DashboardParams) (*DashboardResponse, error) {
	schema, err := schemaForGrupo(ctx, pool, p.GrupoID)
	if err != nil {
		return nil, err
	}

	safe := pgx.Identifier{schema}.Sanitize()
	view := viewName(p.Ano)

	mensal, err := queryGraficoMensal(ctx, pool, safe, view, p)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryDashboard grafico_mensal: %w", err)
	}

	saldoMensal, err := querySaldoMensalContasCorrentes(ctx, pool, safe, p)
	if err != nil {
		return nil, fmt.Errorf("dados.QueryDashboard saldo_cc: %w", err)
	}

	realizadoMes, err := queryRealizadoPorMes(ctx, pool, safe, view, p)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryDashboard realizado_mes: %w", err)
	}

	filtros, err := queryFiltrosDisponiveis(ctx, pool, safe, view, p)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryDashboard filtros: %w", err)
	}

	// Calcula cards a partir dos dados mensais
	var receitaTotal, despesaTotal float64
	for _, m := range mensal {
		receitaTotal += m.Receita
		despesaTotal += m.Despesa
	}

	/*
	 * A âncora só entra quando ela é VERDADE para o recorte.
	 *
	 * O saldo vem do saldo real por mês (saldo_por_contas), que sabe responder
	 * a filtros de empresa e de conta — e só. Com um filtro de categoria,
	 * departamento ou cliente, ancorar no saldo somaria o movimento de um
	 * recorte a uma abertura do total: um número plausível e sem significado.
	 *
	 * Nesse caso a série parte de zero e vira "acumulado do recorte". Continua
	 * verdadeira, só responde outra pergunta — e a resposta diz qual, para a
	 * tela poder mudar o título em vez de mentir no mesmo.
	 *
	 * Vale para as duas séries que partem do saldo real (SaldosMensais e
	 * GraficoResultadoAcumulado) — as duas respondem à mesma pergunta, então
	 * as duas têm que concordar sobre quando a âncora vale.
	 */
	ancoraVale := ancoraAplicavel(p)
	saldoMensalEfetivo := saldoMensal
	if !ancoraVale {
		saldoMensalEfetivo = [12]float64{}
	}
	saldosMensais := acumularSaldos(saldoMensalEfetivo, realizadoMes)

	/*
	 * Atrasados entram só quando pedidos, e só na trilha prevista.
	 *
	 * Falha aqui não derruba o dashboard: sem o acréscimo a projeção fica
	 * otimista, o que é ruim, mas perder a tela inteira por causa disso é pior.
	 * O aviso no log é o que permite descobrir que aconteceu.
	 */
	var atrasados float64
	if p.Inadimplencia {
		if v, errAtr := totalAtrasados(ctx, pool, schema, p); errAtr == nil {
			atrasados = v
		} else {
			log.Warn().Err(errAtr).
				Str("grupo_id", p.GrupoID).
				Msg("dados: atrasados indisponíveis; projeção segue sem eles")
		}
	}
	acumulado := acumularResultado(saldoMensalEfetivo, atrasados, mensal)

	return &DashboardResponse{
		Cards: CardMetrics{
			ReceitaTotal: receitaTotal,
			DespesaTotal: despesaTotal,
			// A fórmula soma posição de caixa a fluxo do período e infla a margem
			// exibida no card — sinalizado ao usuário, aguardando decisão dele.
			// Mantida como estava para não mudar um número em silêncio.
			Resultado:            receitaTotal - despesaTotal + saldosMensais[11].Saldo,
			SaldoContasCorrentes: saldosMensais[11].Saldo,
			SaldoInicial:         saldoMensal[0],
		},
		GraficoMensal:             mensal,
		GraficoResultadoAcumulado: acumulado,
		SaldosMensais:             saldosMensais,
		FiltrosDisponiveis:        filtros,
		AncoraAplicavel:           ancoraVale,
		MesCorte:                  mesCorte(p.Ano),
	}, nil
}

/*
ancoraAplicavel diz se o saldo das contas descreve o mesmo recorte que o fluxo.

O saldo sai de saldo_por_contas (querySaldoMensalContasCorrentes), que conhece
empresa e conta corrente. Categoria, departamento e cliente são dimensões do LANÇAMENTO,
não da conta — não há como perguntar a um extrato bancário quanto do saldo veio
de "Despesas com Pessoal". Com um desses, ancorar somaria o movimento de um
recorte a uma abertura do total.

EXCLUSÃO de categoria é tratada de forma diferente da inclusão, e a distinção é
de intenção: incluir é "me mostre só isto" — um recorte, que invalida a âncora;
excluir é "tire isto do meu fluxo" — uma limpeza, em que o resto continua sendo
o movimento da empresa inteira.

A diferença não é cosmética: a tela exclui transferências por padrão, e tratar
isso como recorte faria o acumulado NUNCA partir do saldo na configuração
normal, que é quando ele mais importa.

O critério é imperfeito por construção — excluir quase todas as categorias tem o
mesmo efeito de um recorte, e aqui passaria como limpeza. Vale pelo caso
dominante, e o custo do erro é uma abertura otimista, não um número inventado.
*/
func ancoraAplicavel(p DashboardParams) bool {
	return len(p.Categorias) == 0 &&
		len(p.Departamentos) == 0 &&
		p.Cliente == ""
}

// buildFiltroAno monta o WHERE completo, usado pelas consultas de dados.
// Extraído para que dashboard e pivot não divirjam: filtro aplicado só num dos
// dois produziria números diferentes para o mesmo recorte, sem erro visível.
func buildFiltroAno(p DashboardParams) (where string, args []any) {
	return buildFiltro(p, nivelTodos)
}

// buildFiltro monta o WHERE aplicando os filtros até o nível pedido. $1 é sempre o ano.
//
// A categoria filtrada é a FINAL, e não a superior. A superior agrupa demais para
// servir de filtro de análise: marcar "Transferência" ali levava junto todas as
// finais abaixo dela. Vale para o filtro, para a exclusão e para a lista de opções
// — as três precisam falar da mesma coluna, senão o usuário marca um nome que o
// WHERE não encontra.
//
// A exclusão de categorias entra apenas em nivelTodos: ela é filtro de dados, não de
// opções. Nas listas a categoria excluída precisa continuar visível para poder ser
// remarcada.
func buildFiltro(p DashboardParams, ate nivelFiltro) (where string, args []any) {
	args = []any{p.Ano}
	conditions := []string{"ano = $1"}
	idx := 2

	add := func(cond string, val any) {
		conditions = append(conditions, fmt.Sprintf(cond, idx))
		args = append(args, val)
		idx++
	}

	/*
	 * A trilha entra em TODOS os níveis, inclusive nos das listas de opções.
	 *
	 * É diferente das outras dimensões, que seguem a cascata para não quebrar o
	 * multi-select. Aqui faz sentido: se o usuário está olhando só o realizado,
	 * uma categoria que só aparece em provisão não deveria ser oferecida como
	 * filtro — ela não existe na tela que ele está vendo.
	 */
	if fontes := trilhasParaFonte(p.Trilhas); fontes != nil {
		add("mov_ou_extrato = ANY($%d)", fontes)
	}

	if ate >= nivelEmpresas && len(p.Empresas) > 0 {
		add("empresa_id::text = ANY($%d)", p.Empresas)
	}
	if ate >= nivelContas && len(p.ContasCorrentes) > 0 {
		add("codigo_conta_corrente = ANY($%d)", p.ContasCorrentes)
	}
	if ate >= nivelDepartamentos && len(p.Departamentos) > 0 {
		add("departamento_final = ANY($%d)", p.Departamentos)
	}
	if ate >= nivelCategorias && len(p.Categorias) > 0 {
		add("descricao_categoria_final = ANY($%d)", p.Categorias)
	}
	if ate >= nivelTodos {
		if len(p.CategoriasExcluir) > 0 {
			add("NOT (COALESCE(descricao_categoria_final, '') = ANY($%d))", p.CategoriasExcluir)
		}
		if p.Cliente != "" {
			add("cliente_final ILIKE $%d", "%"+p.Cliente+"%")
		}
	}

	return strings.Join(conditions, " AND "), args
}

func queryGraficoMensal(ctx context.Context, pool *pgxpool.Pool, safe, view string, p DashboardParams) ([]GraficoMensal, error) {
	where, args := buildFiltroAno(p)
	/*
	 * Receita e despesa saem separadas em REALIZADO e PREVISTO.
	 *
	 * O corte é por mov_ou_extrato, que a view já traz: 'mov' é movimento de
	 * conta corrente — dinheiro que entrou ou saiu — e 'ext' é provisão de
	 * extrato, o que ainda vai acontecer. A fronteira entre os dois é garantida
	 * por considerar_mov_ext na própria view, então não há risco de um
	 * lançamento ser contado nas duas trilhas.
	 *
	 * Os totais continuam somando os dois, para que Receita e Despesa — e tudo
	 * que depende delas — não mudem de valor com esta alteração.
	 */
	sql := fmt.Sprintf(`
		SELECT
			mes,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 1 THEN valor_final ELSE 0 END), 0) AS receita,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 2 THEN valor_final ELSE 0 END), 0) AS despesa,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 1 AND mov_ou_extrato = 'mov' THEN valor_final ELSE 0 END), 0) AS receita_real,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 2 AND mov_ou_extrato = 'mov' THEN valor_final ELSE 0 END), 0) AS despesa_real,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 1 AND mov_ou_extrato = 'ext' THEN valor_final ELSE 0 END), 0) AS receita_prev,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 2 AND mov_ou_extrato = 'ext' THEN valor_final ELSE 0 END), 0) AS despesa_prev
		FROM %s.%s
		WHERE %s
		GROUP BY mes
		ORDER BY mes
	`, safe, view, where)

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Inicializa todos os 12 meses com zero (garante meses sem movimento)
	resultado := make([]GraficoMensal, 12)
	for i := range resultado {
		resultado[i] = GraficoMensal{Mes: i + 1, MesNome: nomeMes[i+1]}
	}

	for rows.Next() {
		var mes int
		var receita, despesa, receitaReal, despesaReal, receitaPrev, despesaPrev float64
		if err := rows.Scan(&mes, &receita, &despesa, &receitaReal, &despesaReal, &receitaPrev, &despesaPrev); err != nil {
			return nil, err
		}
		if mes >= 1 && mes <= 12 {
			m := &resultado[mes-1]
			m.Receita = receita
			m.Despesa = despesa
			m.ResultadoMes = receita - despesa
			m.ReceitaRealizada = receitaReal
			m.DespesaRealizada = despesaReal
			m.ResultadoRealizado = receitaReal - despesaReal
			m.ReceitaPrevista = receitaPrev
			m.DespesaPrevista = despesaPrev
			/*
			 * A regra do Omie, conferida contra um relatório real (ver
			 * acumulado_omie_test.go): o resultado PREVISTO do mês inclui o que
			 * já foi realizado nele. A projeção não é "o que falta", é "como o
			 * mês deve fechar".
			 *
			 * Somar só as provisões daria um número menor e plausível, e a
			 * projeção de caixa ficaria otimista justamente nos meses em que já
			 * houve movimento.
			 */
			m.ResultadoPrevisto = m.ResultadoRealizado + (receitaPrev - despesaPrev)
		}
	}

	return resultado, rows.Err()
}

// querySaldoContasCorrentesCadastro soma contas_correntes.saldo_inicial — campo
// de cadastro do Omie, SEM DATA associada (o cadastro traz essa data em
// raw->>'saldo_data', mas nunca viramos coluna). Usada só como fallback
// transitório, dentro de querySaldoMensalContasCorrentes, para um mês que
// ainda não tem registro em saldo_por_contas (backfill incompleto) — não é
// mais chamada diretamente por QueryDashboard/QueryPivot/QueryFluxoCaixa.
func querySaldoContasCorrentesCadastro(ctx context.Context, pool *pgxpool.Pool, safe string, p DashboardParams) (float64, error) {
	args := []any{p.GrupoID}
	empresaFilter := ""
	if len(p.Empresas) > 0 {
		empresaFilter = " AND e.id::text = ANY($2)"
		args = append(args, p.Empresas)
	}

	// Filtra opcionalmente pelas contas correntes selecionadas
	ccFilter := ""
	nextIdx := len(args) + 1
	if len(p.ContasCorrentes) > 0 {
		ccFilter = fmt.Sprintf(" AND cc.codigo_conta_corrente::text = ANY($%d)", nextIdx)
		args = append(args, p.ContasCorrentes)
	}

	// cc.fluxo_caixa, não cc.raw ->> 'cFluxoCaixa': o campo não existe no cadastro de
	// /geral/contacorrente/ — vem da resposta do ListarExtrato e é propagado pelo
	// executor de extrato. Enquanto lia do raw, este saldo era sempre zero.
	sql := fmt.Sprintf(`
		SELECT COALESCE(SUM(cc.saldo_inicial), 0)
		FROM %s.contas_correntes cc
		JOIN _etl.empresas e ON e.id = cc.empresa_id
		WHERE e.grupo_id = $1
		  AND e.deleted_at IS NULL
		  AND cc.fluxo_caixa = 'S'
		  %s%s
	`, safe, empresaFilter, ccFilter)

	var saldo float64
	err := pool.QueryRow(ctx, sql, args...).Scan(&saldo)
	return saldo, err
}

/*
querySaldoMensalContasCorrentes devolve o saldo REAL de cada mês (índice 0 =
janeiro), somado por conta corrente com fluxo_caixa = 'S', lido de
saldo_por_contas (saldo do dia 1, vindo direto do Omie via ListarExtrato com
cExibirApenasSaldo=S) — não mais acumulado a partir de um saldo de cadastro
sem data.

Aplica os MESMOS filtros que a consulta de cadastro já aplicava (Empresas,
ContasCorrentes — só esses dois fazem sentido para um saldo de conta
corrente; categoria/departamento/cliente são tratados por ancoraAplicavel,
que decide se a âncora participa do recorte, não por esta função).

Fallback por mês: se algum mês ainda não tem linha em saldo_por_contas
(backfill incompleto ou grupo recém-provisionado), usa o saldo de cadastro só
PARA AQUELE MÊS, com aviso no log — evita que o dashboard mostre um mês
isolado zerado enquanto o backfill não terminou.
*/
func querySaldoMensalContasCorrentes(ctx context.Context, pool *pgxpool.Pool, safe string, p DashboardParams) ([12]float64, error) {
	var out [12]float64

	args := []any{p.GrupoID, p.Ano}
	empresaFilter := ""
	if len(p.Empresas) > 0 {
		empresaFilter = fmt.Sprintf(" AND e.id::text = ANY($%d)", len(args)+1)
		args = append(args, p.Empresas)
	}
	ccFilter := ""
	if len(p.ContasCorrentes) > 0 {
		ccFilter = fmt.Sprintf(" AND spc.codigo_conta_corrente::text = ANY($%d)", len(args)+1)
		args = append(args, p.ContasCorrentes)
	}

	sql := fmt.Sprintf(`
		SELECT EXTRACT(MONTH FROM spc.data_referencia)::int AS mes,
		       COALESCE(SUM(spc.saldo_atual), 0) AS saldo
		FROM %s.saldo_por_contas spc
		JOIN %s.contas_correntes cc ON cc.empresa_id = spc.empresa_id
		                            AND cc.codigo_conta_corrente = spc.codigo_conta_corrente
		JOIN _etl.empresas e ON e.id = spc.empresa_id
		WHERE e.grupo_id = $1
		  AND e.deleted_at IS NULL
		  AND cc.fluxo_caixa = 'S'
		  AND EXTRACT(YEAR FROM spc.data_referencia) = $2
		  %s%s
		GROUP BY mes
	`, safe, safe, empresaFilter, ccFilter)

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	var encontrados [12]bool
	for rows.Next() {
		var mes int
		var saldo float64
		if err := rows.Scan(&mes, &saldo); err != nil {
			return out, err
		}
		if mes >= 1 && mes <= 12 {
			out[mes-1] = saldo
			encontrados[mes-1] = true
		}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	var faltaAlgum bool
	for _, ok := range encontrados {
		if !ok {
			faltaAlgum = true
			break
		}
	}
	if faltaAlgum {
		if cadastro, errCad := querySaldoContasCorrentesCadastro(ctx, pool, safe, p); errCad == nil {
			for i, ok := range encontrados {
				if !ok {
					out[i] = cadastro
					log.Warn().Str("grupo_id", p.GrupoID).Int("mes", i+1).
						Msg("dados: saldo real ainda não sincronizado para este mês; usando fallback do cadastro (saldo_inicial)")
				}
			}
		}
	}

	return out, nil
}

/*
queryRealizadoPorMes devolve o resultado REALIZADO de cada mês (índice 0 = janeiro).

Separada de queryGraficoMensal, que soma realizado e previsto juntos: aquele
número responde "quanto entrou e saiu no mês", este responde "quanto disso já
tocou a conta". Saldo em caixa só pode ser construído com o segundo — incluir as
provisões do extrato faria o saldo de dezembro aparecer como dinheiro disponível
hoje.

Usa os mesmos filtros do resto do dashboard (buildFiltroAno), senão o saldo
divergiria dos cards ao lado sob o mesmo recorte.
*/
func queryRealizadoPorMes(ctx context.Context, pool *pgxpool.Pool, safe, view string, p DashboardParams) ([12]float64, error) {
	var out [12]float64

	where, args := buildFiltroAno(p)
	sql := fmt.Sprintf(`
		SELECT
			mes,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 1 THEN valor_final ELSE -valor_final END), 0) AS realizado
		FROM %s.%s
		WHERE %s
		  AND mov_ou_extrato = 'mov'
		GROUP BY mes
	`, safe, view, where)

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	for rows.Next() {
		var mes int
		var valor float64
		if err := rows.Scan(&mes, &valor); err != nil {
			return out, err
		}
		if mes >= 1 && mes <= 12 {
			out[mes-1] = valor
		}
	}
	return out, rows.Err()
}

/*
acumularResultado monta as duas trilhas do modelo do Omie, cada mês
INDEPENDENTE dos demais:

	SaldoInicial(mês) = saldo REAL do mês (saldo_por_contas, dia 1)  [+ atrasados, só na trilha prevista]
	Acumulado(mês)    = SaldoInicial(mês) + Resultado(mês)

Não há mais encadeamento (SaldoInicial(mês) deixou de ser Acumulado(mês-1)):
um mês com saldo real ainda não sincronizado, ou com um erro pontual, não
contamina os meses seguintes — diferente do modelo antigo, que acumulava a
partir de um único saldo de cadastro sem data e propagava qualquer erro de
abertura pelo ano inteiro.

`atrasados` soma em TODOS os meses da trilha prevista, não só em janeiro: no
modelo antigo ele entrava uma única vez na âncora e se propagava para os
meses seguintes por efeito do encadeamento; aqui, sem encadeamento, somar em
todo mês reproduz o mesmo efeito (passivo vencido e não pago que ainda pesa
na projeção), agora sobre uma base real a cada mês. `atrasados` chega zerado
quando o filtro está desligado, e aí as duas trilhas abrem no mesmo ponto.

Pura e sem banco de propósito: é a única aritmética desta parte, e um erro
aqui produz uma série plausível. A tela não denuncia.
*/
func acumularResultado(saldoMensal [12]float64, atrasados float64, mensal []GraficoMensal) []GraficoAcumulado {
	out := make([]GraficoAcumulado, len(mensal))

	for i, m := range mensal {
		saldoIniReal := saldoMensal[i]
		saldoIniPrev := saldoMensal[i] + atrasados

		out[i] = GraficoAcumulado{
			Mes:     m.Mes,
			MesNome: m.MesNome,
			// Campos antigos, preservados byte a byte: a cascata da Visão Geral
			// lê resultado_mes e soma por fora, e trocar isso agora mudaria o
			// desenho antes de a tela estar pronta para as trilhas.
			ResultadoMes: m.ResultadoMes,
			Acumulado:    saldoMensal[i] + m.ResultadoMes,

			SaldoInicialRealizado: saldoIniReal,
			ResultadoRealizado:    m.ResultadoRealizado,
			AcumuladoRealizado:    saldoIniReal + m.ResultadoRealizado,
			SaldoInicialPrevisto:  saldoIniPrev,
			ResultadoPrevisto:     m.ResultadoPrevisto,
			AcumuladoPrevisto:     saldoIniPrev + m.ResultadoPrevisto,
		}
	}
	return out
}

/*
acumularSaldos monta o saldo ao FIM de cada mês: saldo REAL do próprio mês
(saldo_por_contas, dia 1) mais o resultado realizado do mesmo mês.

Função separada e sem banco de propósito: é a única aritmética desta parte, e é
onde um erro passaria despercebido — o número continuaria plausível na tela.

Cada mês é INDEPENDENTE dos demais — não há mais encadeamento (
SaldoInicial(mês) deixou de ser Acumulado(mês-1)). Um mês com saldo real ainda
não sincronizado não contamina os meses seguintes, diferente do modelo antigo
baseado em contas_correntes.saldo_inicial (cadastro do Omie, sem data).
*/
func acumularSaldos(saldoMensal, realizadoPorMes [12]float64) []SaldoMes {
	out := make([]SaldoMes, 12)
	for i := 0; i < 12; i++ {
		out[i] = SaldoMes{Mes: i + 1, MesNome: nomeMes[i+1], Saldo: saldoMensal[i] + realizadoPorMes[i]}
	}
	return out
}

// queryFiltrosDisponiveis monta as opções de cada filtro em CASCATA: cada lista é
// restringida apenas pelos filtros de nível superior.
//
// Restringir um filtro por si mesmo quebraria o multi-select — marcar uma categoria
// faria as demais desaparecerem da lista, impedindo marcar a segunda.
func queryFiltrosDisponiveis(ctx context.Context, pool *pgxpool.Pool, safe, view string, p DashboardParams) (FiltrosDisponiveis, error) {
	var f FiltrosDisponiveis

	// Empresas: só o ano restringe. Vem de _etl.empresas e não da view, para que uma
	// empresa recém-cadastrada apareça no filtro antes de ter movimento sincronizado.
	rowsEmp, err := pool.Query(ctx, `
		SELECT id::text, nome
		FROM _etl.empresas
		WHERE grupo_id = $1 AND deleted_at IS NULL
		ORDER BY nome
	`, p.GrupoID)
	if err != nil {
		return f, fmt.Errorf("filtros empresas: %w", err)
	}
	f.Empresas = []EmpresaItem{}
	for rowsEmp.Next() {
		var item EmpresaItem
		if err := rowsEmp.Scan(&item.ID, &item.Nome); err != nil {
			rowsEmp.Close()
			return f, err
		}
		f.Empresas = append(f.Empresas, item)
	}
	rowsEmp.Close()
	if err := rowsEmp.Err(); err != nil {
		return f, fmt.Errorf("filtros empresas: %w", err)
	}

	// Contas correntes: restringidas pelas empresas selecionadas. Vêm da tabela de
	// cadastro, não da view, para incluir conta sem movimento no período.
	//
	// Sem corte por fluxo_caixa: a lista traz TODAS as contas. Antes só saíam as
	// marcadas 'S', o que escondia contas cujos movimentos realizados já entravam
	// nos números — o usuário não conseguia nem vê-las, nem desmarcá-las. A marca
	// agora acompanha o item e serve só para agrupar o filtro na tela.
	ccArgs := []any{p.GrupoID}
	ccEmpresaFilter := ""
	if len(p.Empresas) > 0 {
		ccEmpresaFilter = " AND e.id::text = ANY($2)"
		ccArgs = append(ccArgs, p.Empresas)
	}
	rowsCC, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT cc.codigo_conta_corrente::text, cc.descricao,
		       COALESCE(cc.fluxo_caixa, '')
		FROM %s.contas_correntes cc
		JOIN _etl.empresas e ON e.id = cc.empresa_id
		WHERE e.grupo_id = $1
		  AND e.deleted_at IS NULL%s
		ORDER BY cc.descricao
	`, safe, ccEmpresaFilter), ccArgs...)
	if err != nil {
		return f, fmt.Errorf("filtros contas_correntes: %w", err)
	}
	f.ContasCorrentes = []ContaCorrenteItem{}
	for rowsCC.Next() {
		var item ContaCorrenteItem
		if err := rowsCC.Scan(&item.Codigo, &item.Descricao, &item.FluxoCaixa); err != nil {
			rowsCC.Close()
			return f, err
		}
		f.ContasCorrentes = append(f.ContasCorrentes, item)
	}
	rowsCC.Close()
	if err := rowsCC.Err(); err != nil {
		return f, fmt.Errorf("filtros contas_correntes: %w", err)
	}

	// Os três seguintes saem da view, cada um com o WHERE do seu nível.
	textos := []struct {
		coluna string
		nivel  nivelFiltro
		limite string
		dest   *[]string
		nome   string
	}{
		{"departamento_final", nivelContas, "", &f.Departamentos, "departamentos"},
		{"descricao_categoria_final", nivelDepartamentos, "", &f.Categorias, "categorias"},
		{"cliente_final", nivelCategorias, "LIMIT 300", &f.Clientes, "clientes"},
	}

	for _, t := range textos {
		where, args := buildFiltro(p, t.nivel)
		sql := fmt.Sprintf(`
			SELECT DISTINCT %s
			FROM %s.%s
			WHERE %s AND %s IS NOT NULL AND %s != ''
			ORDER BY %s
			%s
		`, t.coluna, safe, view, where, t.coluna, t.coluna, t.coluna, t.limite)

		rows, err := pool.Query(ctx, sql, args...)
		if err != nil {
			return f, fmt.Errorf("filtros %s: %w", t.nome, err)
		}
		*t.dest = []string{}
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				rows.Close()
				return f, fmt.Errorf("filtros %s scan: %w", t.nome, err)
			}
			*t.dest = append(*t.dest, v)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return f, fmt.Errorf("filtros %s: %w", t.nome, err)
		}
	}

	return f, nil
}

// QueryFiltros devolve apenas as opções de filtro, sem a agregação de dados.
func QueryFiltros(ctx context.Context, pool *pgxpool.Pool, p DashboardParams) (*FiltrosDisponiveis, error) {
	schema, err := schemaForGrupo(ctx, pool, p.GrupoID)
	if err != nil {
		return nil, err
	}

	f, err := queryFiltrosDisponiveis(ctx, pool, pgx.Identifier{schema}.Sanitize(), viewName(p.Ano), p)
	if err != nil {
		if isViewNaoPopulada(err) {
			return nil, ErrViewNaoPopulada
		}
		return nil, fmt.Errorf("dados.QueryFiltros: %w", err)
	}
	return &f, nil
}
