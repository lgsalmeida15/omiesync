package dados

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	pool   *pgxpool.Pool
	jwtSvc auth.JWTService
}

func NewHandler(pool *pgxpool.Pool, jwtSvc auth.JWTService) *Handler {
	return &Handler{pool: pool, jwtSvc: jwtSvc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))

	r.Get("/{empresaID}/clientes", h.queryHandler("clientes",
		"SELECT id, codigo_cliente_omie, razao_social, nome_fantasia, cnpj_cpf, email, cidade, estado, ativo, synced_at FROM %s.clientes ORDER BY razao_social"))

	r.Get("/{empresaID}/categorias", h.queryHandler("categorias",
		"SELECT id, codigo, descricao, synced_at FROM %s.categorias ORDER BY codigo"))

	r.Get("/{empresaID}/departamentos", h.queryHandler("departamentos",
		"SELECT id, codigo, descricao, inativo, synced_at FROM %s.departamentos ORDER BY codigo"))

	r.Get("/{empresaID}/contas-correntes", h.queryHandler("contas_correntes",
		"SELECT id, codigo_conta_corrente, descricao, tipo, saldo_inicial, synced_at FROM %s.contas_correntes ORDER BY descricao"))

	r.Get("/{empresaID}/contas-pagar", h.queryHandler("contas_pagar",
		"SELECT id, codigo_lancamento, data_vencimento, valor_documento, valor_pago, status_titulo, codigo_cliente, observacao, synced_at FROM %s.contas_pagar ORDER BY data_vencimento DESC"))

	r.Get("/{empresaID}/contas-receber", h.queryHandler("contas_receber",
		"SELECT id, codigo_lancamento, data_vencimento, valor_documento, valor_recebido, status_titulo, codigo_cliente, observacao, synced_at FROM %s.contas_receber ORDER BY data_vencimento DESC"))

	r.Get("/{empresaID}/movimentos", h.queryHandler("movimentos_financeiros",
		"SELECT id, codigo_titulo, numero_titulo, codigo_mov_cc, data_lancamento, valor, tipo, codigo_conta_corrente, codigo_categoria, synced_at FROM %s.movimentos_financeiros ORDER BY data_lancamento DESC"))

	r.Get("/{empresaID}/extrato", h.queryHandler("extrato",
		"SELECT id, codigo_lancamento, data_lancamento, valor, tipo_lancamento, codigo_conta_corrente, descricao, synced_at FROM %s.extrato ORDER BY data_lancamento DESC"))

	r.Get("/{empresaID}/ordens-servico", h.queryHandler("ordens_servico",
		"SELECT id, numero_os, data_abertura, data_previsao, data_fechamento, status, codigo_cliente, valor_total, synced_at FROM %s.ordens_servico ORDER BY data_abertura DESC"))

	r.Get("/{empresaID}/projetos", h.queryHandler("projetos",
		"SELECT id, codigo_projeto, nome, descricao, data_inicio, data_fim, status, synced_at FROM %s.projetos ORDER BY nome"))

	r.Get("/{grupoID}/dashboard", h.dashboardHandler)
	r.Get("/{grupoID}/pivot", h.pivotHandler)
	r.Get("/{grupoID}/filtros", h.filtrosHandler)

	return r
}

// queryHandler retorna um http.HandlerFunc genérico que executa uma query no schema do tenant.
func (h *Handler) queryHandler(tabela, sqlTemplate string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		empresaID := chi.URLParam(r, "empresaID")

		schema, grupoID, err := h.schemaForEmpresa(r.Context(), empresaID)
		if err != nil {
			response.NotFound(w, "empresa não encontrada ou sem dados sincronizados")
			return
		}

		// Verifica que o usuário pertence ao grupo dono da empresa (previne IDOR)
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			response.Unauthorized(w, "não autenticado")
			return
		}
		if claims.Role != "admin_global" && claims.GrupoID != grupoID {
			ae := apperror.Forbidden("acesso negado")
			response.FromAppError(w, ae)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		if page <= 0 {
			page = 1
		}
		if perPage <= 0 || perPage > 500 {
			perPage = 100
		}

		offset := (page - 1) * perPage
		sql := fmt.Sprintf(sqlTemplate+" LIMIT $1 OFFSET $2", schema)

		rows, err := h.pool.Query(r.Context(), sql, perPage, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "erro ao consultar dados", err)
			return
		}
		defer rows.Close()

		// Constrói array de mapas dinamicamente
		cols := rows.FieldDescriptions()
		var result []map[string]any

		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "erro ao ler linha", err)
				return
			}
			row := make(map[string]any, len(cols))
			for i, col := range cols {
				row[string(col.Name)] = vals[i]
			}
			result = append(result, row)
		}

		if result == nil {
			result = []map[string]any{}
		}

		// Conta total
		var total int64
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, tabela)
		_ = h.pool.QueryRow(r.Context(), countSQL).Scan(&total)

		response.OKPaginated(w, result, response.Meta{
			Page: page, PerPage: perPage, Total: int(total),
		})
	}
}

// autorizaGrupo garante que o usuário só leia dados do próprio grupo.
// admin_global acessa qualquer um. Retorna false já tendo escrito a resposta.
func autorizaGrupo(w http.ResponseWriter, r *http.Request, grupoID string) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return false
	}
	if claims.Role != "admin_global" && claims.GrupoID != grupoID {
		response.FromAppError(w, apperror.Forbidden("acesso negado"))
		return false
	}
	return true
}

// parseDashboardParams lê os filtros da query string. Compartilhado entre dashboard
// e pivot: divergência aqui faria as duas telas mostrarem recortes diferentes do
// mesmo período, sem nenhum erro aparente.
func parseDashboardParams(r *http.Request, grupoID string) DashboardParams {
	ano := time.Now().Year()
	if v := r.URL.Query().Get("ano"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 2000 {
			ano = parsed
		}
	}

	split := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		return out
	}

	return DashboardParams{
		GrupoID:         grupoID,
		Ano:             ano,
		Empresas:        split(r.URL.Query().Get("empresas")),
		ContasCorrentes: split(r.URL.Query().Get("contas_correntes")),
		Departamentos:   split(r.URL.Query().Get("departamentos")),
		Categorias:      split(r.URL.Query().Get("categorias")),
		Cliente:         strings.TrimSpace(r.URL.Query().Get("cliente")),
		// Vazio por padrão: o backend não decide o que ocultar. Quem define o
		// padrão é a tela, para que outros consumidores da API não herdem a
		// exclusão sem pedir.
		CategoriasExcluir: split(r.URL.Query().Get("categorias_excluir")),
	}
}

// filtrosHandler serve apenas as opções de filtro, em cascata.
//
// Separado do dashboard porque as cinco consultas DISTINCT eram reexecutadas junto
// com a agregação pesada a cada mudança de filtro. Agora a tela recarrega as opções
// só quando um filtro de nível superior muda.
func (h *Handler) filtrosHandler(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	if !autorizaGrupo(w, r, grupoID) {
		return
	}

	data, err := QueryFiltros(r.Context(), h.pool, parseDashboardParams(r, grupoID))
	if err != nil {
		if errors.Is(err, ErrViewNaoPopulada) {
			response.FromAppError(w, apperror.Unprocessable(ErrViewNaoPopulada.Error()))
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao consultar filtros", err)
		return
	}
	response.OK(w, data)
}

func (h *Handler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	if !autorizaGrupo(w, r, grupoID) {
		return
	}

	params := parseDashboardParams(r, grupoID)

	data, err := QueryDashboard(r.Context(), h.pool, params)
	if err != nil {
		// View recriada e ainda sem REFRESH não é falha do servidor — é estado
		// transitório após um re-provisionamento. Devolve mensagem acionável em vez
		// de "erro ao consultar dashboard".
		if errors.Is(err, ErrViewNaoPopulada) {
			response.FromAppError(w, apperror.Unprocessable(ErrViewNaoPopulada.Error()))
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao consultar dashboard", err)
		return
	}

	response.OK(w, data)
}

// pivotHandler serve a aba "Resultado": matriz de categoria e cliente por mês.
func (h *Handler) pivotHandler(w http.ResponseWriter, r *http.Request) {
	grupoID := chi.URLParam(r, "grupoID")
	if !autorizaGrupo(w, r, grupoID) {
		return
	}

	data, err := QueryPivot(r.Context(), h.pool, parseDashboardParams(r, grupoID))
	if err != nil {
		if errors.Is(err, ErrViewNaoPopulada) {
			response.FromAppError(w, apperror.Unprocessable(ErrViewNaoPopulada.Error()))
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao consultar resultado", err)
		return
	}

	response.OK(w, data)
}

func (h *Handler) schemaForEmpresa(ctx context.Context, empresaID string) (schema, grupoID string, err error) {
	err = h.pool.QueryRow(ctx, `
		SELECT g.schema_name, e.grupo_id
		FROM _etl.empresas e
		JOIN _etl.grupos g ON g.id = e.grupo_id
		WHERE e.id = $1 AND e.deleted_at IS NULL
	`, empresaID).Scan(&schema, &grupoID)
	if err != nil {
		return "", "", fmt.Errorf("dados.schemaForEmpresa: %w", err)
	}
	return schema, grupoID, nil
}
