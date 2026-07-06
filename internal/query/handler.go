package query

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

// Handler expõe o endpoint do SQL Explorer.
type Handler struct {
	svc    Service
	pool   *pgxpool.Pool
	jwtSvc auth.JWTService
}

// NewHandler cria um Handler do SQL Explorer.
func NewHandler(svc Service, pool *pgxpool.Pool, jwtSvc auth.JWTService) *Handler {
	return &Handler{svc: svc, pool: pool, jwtSvc: jwtSvc}
}

// Execute processa POST /admin/grupos/{grupoID}/query.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	grupoID := chi.URLParam(r, "grupoID")

	// admin_grupo só pode consultar o próprio grupo
	if claims.Role == "admin_grupo" && claims.GrupoID != grupoID {
		response.Forbidden(w, "acesso negado a este grupo")
		return
	}

	// Busca o schema_name do grupo diretamente no banco
	var schemaName string
	err := h.pool.QueryRow(
		r.Context(),
		"SELECT schema_name FROM _etl.grupos WHERE id = $1 AND deleted_at IS NULL",
		grupoID,
	).Scan(&schemaName)
	if err != nil {
		response.NotFound(w, "grupo não encontrado")
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	if err := h.svc.ValidateSQL(req.SQL); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusBadRequest, "SQL inválido", err)
		return
	}

	result, err := h.svc.Execute(r.Context(), h.pool, schemaName, req.SQL)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao executar query", err)
		return
	}

	response.OK(w, result)
}
