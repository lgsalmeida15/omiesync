package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/omie"
)

type listarClientesResp struct {
	omie.PaginacaoResponse
	ClientesCadastro []OmieCliente `json:"clientes_cadastro"`
}

type listarContasPagarResp struct {
	omie.PaginacaoResponse
	ContaPagarCadastro []OmieContaPagar `json:"conta_pagar_cadastro"`
}

// omieServer simula a API do Omie retornando uma página de dados.
func omieServer(t *testing.T, respBody any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
}


func TestClientesExecutor_Nome(t *testing.T) {
	e := &ClientesExecutor{}
	if e.Nome() != "clientes" {
		t.Errorf("nome: got %q want clientes", e.Nome())
	}
}

func TestAllExecutors_NomesUnicos(t *testing.T) {
	execs := NewAllExecutors(nil, zerolog.Nop())
	nomes := make(map[string]bool)
	for _, e := range execs {
		if nomes[e.Nome()] {
			t.Errorf("executor com nome duplicado: %q", e.Nome())
		}
		nomes[e.Nome()] = true
	}
	if len(execs) != 10 {
		t.Errorf("esperava 10 executors, got %d", len(execs))
	}
}

func TestAllExecutors_TodosOsModulos(t *testing.T) {
	esperados := []string{
		"categorias", "departamentos", "contas_correntes",
		"clientes", "contas_pagar", "contas_receber",
		"movimentos_financeiros", "extrato", "ordens_servico", "projetos",
	}
	execs := NewAllExecutors(nil, zerolog.Nop())
	nomes := make(map[string]bool)
	for _, e := range execs {
		nomes[e.Nome()] = true
	}
	for _, esperado := range esperados {
		if !nomes[esperado] {
			t.Errorf("executor %q nÃ£o encontrado", esperado)
		}
	}
}

func TestClientesExecutor_ParseResponse(t *testing.T) {
	respBody := listarClientesResp{
		PaginacaoResponse: omie.PaginacaoResponse{
			Pagina: 1, TotalDePaginas: 1, Registros: 2, TotalDeRegistros: 2,
		},
		ClientesCadastro: []OmieCliente{
			{CodigoClienteOmie: 1001, RazaoSocial: "Empresa A", CnpjCpf: "12.345.678/0001-99"},
			{CodigoClienteOmie: 1002, RazaoSocial: "Empresa B", CnpjCpf: "98.765.432/0001-11"},
		},
	}

	srv := omieServer(t, respBody)
	defer srv.Close()

	c := clientForServer(srv)
	var result listarClientesResp
	err := c.CallPublic(context.Background(), "/geral/clientes/", "ListarClientes",
		omie.PaginacaoParams{Pagina: 1, RegistrosPorPagina: 50}, &result)

	if err != nil {
		t.Fatalf("CallPublic: %v", err)
	}
	if len(result.ClientesCadastro) != 2 {
		t.Errorf("clientes: got %d want 2", len(result.ClientesCadastro))
	}
	if result.ClientesCadastro[0].RazaoSocial != "Empresa A" {
		t.Errorf("razao_social: got %q", result.ClientesCadastro[0].RazaoSocial)
	}
}

func TestContasPagarExecutor_ParseResponse(t *testing.T) {
	respBody := listarContasPagarResp{
		PaginacaoResponse: omie.PaginacaoResponse{Pagina: 1, TotalDePaginas: 1},
		ContaPagarCadastro: []OmieContaPagar{
			{CodigoLancamento: 5001, ValorDocumento: 1500.00, StatusTitulo: "ABERTO"},
		},
	}
	srv := omieServer(t, respBody)
	defer srv.Close()
	c := clientForServer(srv)

	var result listarContasPagarResp
	err := c.CallPublic(context.Background(), "/financas/contapagar/", "ListarContasPagar",
		omie.PaginacaoParams{Pagina: 1, RegistrosPorPagina: 50}, &result)

	if err != nil {
		t.Fatalf("CallPublic: %v", err)
	}
	if result.ContaPagarCadastro[0].ValorDocumento != 1500.00 {
		t.Errorf("valor: got %f want 1500.00", result.ContaPagarCadastro[0].ValorDocumento)
	}
}

func TestToJSON(t *testing.T) {
	v := map[string]any{"key": "value", "num": 42}
	b := toJSON(v)
	if len(b) == 0 {
		t.Error("toJSON retornou vazio")
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Errorf("toJSON output invÃ¡lido: %v", err)
	}
}

// Verifica que o pool nil nÃ£o causa panic na criaÃ§Ã£o dos executors
func TestNewAllExecutors_NilPool(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic ao criar executors com pool nil: %v", r)
		}
	}()
	execs := NewAllExecutors((*pgxpool.Pool)(nil), zerolog.Nop())
	if len(execs) == 0 {
		t.Error("nenhum executor criado")
	}
	_ = fmt.Sprintf("%d executors", len(execs))
}

