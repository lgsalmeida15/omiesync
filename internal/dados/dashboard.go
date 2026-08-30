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

	saldo, err := querySaldoContasCorrentes(ctx, pool, safe, p)
	if err != nil {
		return nil, fmt.Errorf("dados.QueryDashboard saldo_cc: %w", err)
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

	// Monta grafico_resultado_acumulado
	acumulado := make([]GraficoAcumulado, len(mensal))
	var acc float64
	for i, m := range mensal {
		acc += m.ResultadoMes
		acumulado[i] = GraficoAcumulado{
			Mes:          m.Mes,
			MesNome:      m.MesNome,
			ResultadoMes: m.ResultadoMes,
			Acumulado:    acc,
		}
	}

	return &DashboardResponse{
		Cards: CardMetrics{
			ReceitaTotal:         receitaTotal,
			DespesaTotal:         despesaTotal,
			Resultado:            receitaTotal - despesaTotal + saldo,
			SaldoContasCorrentes: saldo,
		},
		GraficoMensal:             mensal,
		GraficoResultadoAcumulado: acumulado,
		FiltrosDisponiveis:        filtros,
	}, nil
}

// buildFiltroAno monta o WHERE completo, usado pelas consultas de dados.
// Extraído para que dashboard e pivot não divirjam: filtro aplicado só num dos
// dois produziria números diferentes para o mesmo recorte, sem erro visível.
func buildFiltroAno(p DashboardParams) (where string, args []any) {
	return buildFiltro(p, nivelTodos)
}

// buildFiltro monta o WHERE aplicando os filtros até o nível pedido. $1 é sempre o ano.
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
		add("descricao_categoria_superior = ANY($%d)", p.Categorias)
	}
	if ate >= nivelTodos {
		if len(p.CategoriasExcluir) > 0 {
			add("NOT (COALESCE(descricao_categoria_superior, '') = ANY($%d))", p.CategoriasExcluir)
		}
		if p.Cliente != "" {
			add("cliente_final ILIKE $%d", "%"+p.Cliente+"%")
		}
	}

	return strings.Join(conditions, " AND "), args
}

func queryGraficoMensal(ctx context.Context, pool *pgxpool.Pool, safe, view string, p DashboardParams) ([]GraficoMensal, error) {
	where, args := buildFiltroAno(p)
	sql := fmt.Sprintf(`
		SELECT
			mes,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 1 THEN valor_final ELSE 0 END), 0) AS receita,
			COALESCE(SUM(CASE WHEN ajuste_receita_despesa = 2 THEN valor_final ELSE 0 END), 0) AS despesa
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
		var receita, despesa float64
		if err := rows.Scan(&mes, &receita, &despesa); err != nil {
			return nil, err
		}
		if mes >= 1 && mes <= 12 {
			resultado[mes-1].Receita = receita
			resultado[mes-1].Despesa = despesa
			resultado[mes-1].ResultadoMes = receita - despesa
		}
	}

	return resultado, rows.Err()
}

func querySaldoContasCorrentes(ctx context.Context, pool *pgxpool.Pool, safe string, p DashboardParams) (float64, error) {
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
		{"descricao_categoria_superior", nivelDepartamentos, "", &f.Categorias, "categorias"},
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
