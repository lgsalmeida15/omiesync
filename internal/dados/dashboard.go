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
}

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

	filtros, err := queryFiltrosDisponiveis(ctx, pool, safe, view, p.GrupoID)
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

func queryGraficoMensal(ctx context.Context, pool *pgxpool.Pool, safe, view string, p DashboardParams) ([]GraficoMensal, error) {
	args := []any{p.Ano}
	conditions := []string{"ano = $1"}
	idx := 2

	if len(p.Empresas) > 0 {
		conditions = append(conditions, fmt.Sprintf("empresa_id::text = ANY($%d)", idx))
		args = append(args, p.Empresas)
		idx++
	}
	if len(p.ContasCorrentes) > 0 {
		conditions = append(conditions, fmt.Sprintf("codigo_conta_corrente = ANY($%d)", idx))
		args = append(args, p.ContasCorrentes)
		idx++
	}
	if len(p.Departamentos) > 0 {
		conditions = append(conditions, fmt.Sprintf("departamento_final = ANY($%d)", idx))
		args = append(args, p.Departamentos)
		idx++
	}
	if len(p.Categorias) > 0 {
		conditions = append(conditions, fmt.Sprintf("descricao_categoria_superior = ANY($%d)", idx))
		args = append(args, p.Categorias)
		idx++
	}
	if p.Cliente != "" {
		conditions = append(conditions, fmt.Sprintf("cliente_final ILIKE $%d", idx))
		args = append(args, "%"+p.Cliente+"%")
	}

	where := strings.Join(conditions, " AND ")
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

func queryFiltrosDisponiveis(ctx context.Context, pool *pgxpool.Pool, safe, view, grupoID string) (FiltrosDisponiveis, error) {
	var f FiltrosDisponiveis

	// Contas correntes do grupo (com nome)
	rowsCC, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT cc.codigo_conta_corrente::text, cc.descricao
		FROM %s.contas_correntes cc
		JOIN _etl.empresas e ON e.id = cc.empresa_id
		WHERE e.grupo_id = $1
		  AND e.deleted_at IS NULL
		  AND cc.fluxo_caixa = 'S'
		ORDER BY cc.descricao
	`, safe), grupoID)
	if err != nil {
		return f, fmt.Errorf("filtros contas_correntes: %w", err)
	}
	defer rowsCC.Close()
	for rowsCC.Next() {
		var item ContaCorrenteItem
		if err := rowsCC.Scan(&item.Codigo, &item.Descricao); err != nil {
			return f, err
		}
		f.ContasCorrentes = append(f.ContasCorrentes, item)
	}
	if f.ContasCorrentes == nil {
		f.ContasCorrentes = []ContaCorrenteItem{}
	}

	// Departamentos distintos da view
	rowsDept, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT departamento_final
		FROM %s.%s
		WHERE departamento_final IS NOT NULL AND departamento_final != ''
		ORDER BY departamento_final
	`, safe, view))
	if err != nil {
		return f, fmt.Errorf("filtros departamentos: %w", err)
	}
	defer rowsDept.Close()
	for rowsDept.Next() {
		var d string
		if err := rowsDept.Scan(&d); err != nil {
			return f, err
		}
		f.Departamentos = append(f.Departamentos, d)
	}
	if f.Departamentos == nil {
		f.Departamentos = []string{}
	}

	// Categorias superiores distintas da view
	rowsCat, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT descricao_categoria_superior
		FROM %s.%s
		WHERE descricao_categoria_superior IS NOT NULL AND descricao_categoria_superior != ''
		ORDER BY descricao_categoria_superior
	`, safe, view))
	if err != nil {
		return f, fmt.Errorf("filtros categorias: %w", err)
	}
	defer rowsCat.Close()
	for rowsCat.Next() {
		var c string
		if err := rowsCat.Scan(&c); err != nil {
			return f, err
		}
		f.Categorias = append(f.Categorias, c)
	}
	if f.Categorias == nil {
		f.Categorias = []string{}
	}

	// Clientes distintos da view (para autocomplete)
	rowsCli, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT cliente_final
		FROM %s.%s
		WHERE cliente_final IS NOT NULL AND cliente_final != ''
		ORDER BY cliente_final
		LIMIT 300
	`, safe, view))
	if err != nil {
		return f, fmt.Errorf("filtros clientes: %w", err)
	}
	defer rowsCli.Close()
	for rowsCli.Next() {
		var c string
		if err := rowsCli.Scan(&c); err != nil {
			return f, err
		}
		f.Clientes = append(f.Clientes, c)
	}
	if f.Clientes == nil {
		f.Clientes = []string{}
	}

	// Empresas ativas do grupo
	rowsEmp, err := pool.Query(ctx, `
		SELECT id::text, nome
		FROM _etl.empresas
		WHERE grupo_id = $1 AND deleted_at IS NULL
		ORDER BY nome
	`, grupoID)
	if err != nil {
		return f, fmt.Errorf("filtros empresas: %w", err)
	}
	defer rowsEmp.Close()
	for rowsEmp.Next() {
		var item EmpresaItem
		if err := rowsEmp.Scan(&item.ID, &item.Nome); err != nil {
			return f, err
		}
		f.Empresas = append(f.Empresas, item)
	}
	if f.Empresas == nil {
		f.Empresas = []EmpresaItem{}
	}

	return f, nil
}
