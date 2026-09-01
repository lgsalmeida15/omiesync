package dados

import (
	"strings"
	"testing"
)

// buildFiltro alimenta TODAS as abas do dashboard — visão geral, resultado, fluxo
// de caixa e as duas de contas. Um erro aqui não quebra a tela: ela mostra números
// errados em silêncio, que é pior. Daí o teste.

func TestBuildFiltro_AnoSempreEmPrimeiro(t *testing.T) {
	where, args := buildFiltro(DashboardParams{Ano: 2026}, nivelTodos)

	if !strings.Contains(where, "ano = $1") {
		t.Errorf("ano deveria ser o primeiro placeholder; where = %q", where)
	}
	if len(args) != 1 || args[0] != 2026 {
		t.Errorf("args = %v, esperado [2026]", args)
	}
}

// A categoria filtrada é a FINAL. A superior agrupa demais: marcar "Transferência"
// levava junto todas as finais abaixo dela.
func TestBuildFiltro_CategoriaUsaFinalNaoSuperior(t *testing.T) {
	where, _ := buildFiltro(DashboardParams{
		Ano:        2026,
		Categorias: []string{"Vendas"},
	}, nivelTodos)

	if !strings.Contains(where, "descricao_categoria_final") {
		t.Errorf("filtro deveria usar a categoria final; where = %q", where)
	}
	if strings.Contains(where, "descricao_categoria_superior") {
		t.Errorf("filtro não deveria mais citar a categoria superior; where = %q", where)
	}
}

func TestBuildFiltro_ExclusaoUsaFinalENegativa(t *testing.T) {
	where, args := buildFiltro(DashboardParams{
		Ano:               2026,
		CategoriasExcluir: []string{"Transf. Inter Empresas/Contas"},
	}, nivelTodos)

	if !strings.Contains(where, "NOT (COALESCE(descricao_categoria_final") {
		t.Errorf("exclusão deveria negar sobre a categoria final; where = %q", where)
	}
	// COALESCE porque categoria nula não pode escapar da exclusão por ser nula.
	if !strings.Contains(where, "COALESCE") {
		t.Errorf("exclusão precisa de COALESCE para alcançar nulos; where = %q", where)
	}
	if len(args) != 2 {
		t.Errorf("esperado ano + lista de exclusão, veio %d args", len(args))
	}
}

// A cascata existe para que marcar um filtro não esvazie a lista do seguinte. Cada
// nível só aplica os filtros anteriores a ele.
func TestBuildFiltro_CascataRespeitaNivel(t *testing.T) {
	p := DashboardParams{
		Ano:             2026,
		Empresas:        []string{"e1"},
		ContasCorrentes: []string{"101"},
		Departamentos:   []string{"D1"},
		Categorias:      []string{"Vendas"},
	}

	casos := []struct {
		nivel     nivelFiltro
		contem    []string
		naoContem []string
	}{
		{nivelAno, nil, []string{"empresa_id", "codigo_conta_corrente", "departamento_final", "descricao_categoria_final"}},
		{nivelEmpresas, []string{"empresa_id"}, []string{"codigo_conta_corrente", "departamento_final"}},
		{nivelContas, []string{"empresa_id", "codigo_conta_corrente"}, []string{"departamento_final"}},
		{nivelDepartamentos, []string{"departamento_final"}, []string{"descricao_categoria_final"}},
		{nivelCategorias, []string{"descricao_categoria_final"}, nil},
	}

	for _, c := range casos {
		where, _ := buildFiltro(p, c.nivel)
		for _, esperado := range c.contem {
			if !strings.Contains(where, esperado) {
				t.Errorf("nivel %d deveria conter %q; where = %q", c.nivel, esperado, where)
			}
		}
		for _, proibido := range c.naoContem {
			if strings.Contains(where, proibido) {
				t.Errorf("nivel %d não deveria conter %q; where = %q", c.nivel, proibido, where)
			}
		}
	}
}

// Placeholders fora de ordem fazem o pgx casar argumento com a coluna errada — e o
// resultado é filtro silenciosamente trocado, não erro.
func TestBuildFiltro_PlaceholdersSequenciais(t *testing.T) {
	_, args := buildFiltro(DashboardParams{
		Ano:             2026,
		Empresas:        []string{"e1"},
		ContasCorrentes: []string{"101"},
		Departamentos:   []string{"D1"},
		Categorias:      []string{"Vendas"},
		Cliente:         "ACME",
	}, nivelTodos)

	if len(args) != 6 {
		t.Fatalf("esperado 6 args (ano + 5 filtros), veio %d", len(args))
	}
	where, _ := buildFiltro(DashboardParams{
		Ano:      2026,
		Empresas: []string{"e1"},
	}, nivelTodos)
	if !strings.Contains(where, "$2") {
		t.Errorf("segundo filtro deveria usar $2; where = %q", where)
	}
}

// Cliente é busca parcial: filtrar por igualdade obrigaria o nome exato.
func TestBuildFiltro_ClienteUsaILIKEComCuringas(t *testing.T) {
	where, args := buildFiltro(DashboardParams{Ano: 2026, Cliente: "ACME"}, nivelTodos)

	if !strings.Contains(where, "cliente_final ILIKE") {
		t.Errorf("cliente deveria usar ILIKE; where = %q", where)
	}
	if args[1] != "%ACME%" {
		t.Errorf("cliente deveria vir entre curingas, veio %v", args[1])
	}
}

// Lista vazia não pode virar `= ANY('{}')`, que não casa com nada e esvaziaria a tela.
func TestBuildFiltro_ListasVaziasNaoEntram(t *testing.T) {
	where, args := buildFiltro(DashboardParams{
		Ano:             2026,
		Empresas:        []string{},
		ContasCorrentes: []string{},
		Categorias:      []string{},
	}, nivelTodos)

	if where != "ano = $1" {
		t.Errorf("sem filtros o where deveria ser só o ano; veio %q", where)
	}
	if len(args) != 1 {
		t.Errorf("esperado só o ano, veio %d args", len(args))
	}
}
