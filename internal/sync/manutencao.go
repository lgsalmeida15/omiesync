package sync

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/response"
)

// Ações operacionais de banco, restritas a admin_global.
//
// Existem como endpoints dedicados, e não como SQL livre no SQL Explorer, porque
// têm alcance real: o REFRESH pode segurar a view por minutos e o cancelamento
// derruba consultas em andamento. Botão com alvo explícito erra menos que SQL
// digitado à mão, e cada chamada passa pelo middleware de auditoria.

// ConsultaAtiva descreve uma consulta em execução no banco.
type ConsultaAtiva struct {
	PID          int32   `json:"pid"`
	Estado       string  `json:"estado"`
	EsperandoPor *string `json:"esperando_por"`
	DuracaoSeg   float64 `json:"duracao_seg"`
	Query        string  `json:"query"`
	Aplicacao    string  `json:"aplicacao"`
}

// RefreshViewResultado relata o desfecho da atualização de uma view.
type RefreshViewResultado struct {
	View string `json:"view"`
	// Concorrente é false quando a view ainda não estava populada e foi preciso
	// usar o modo bloqueante — o único caso em que o dashboard fica indisponível.
	Concorrente bool    `json:"concorrente"`
	DuracaoSeg  float64 `json:"duracao_seg"`
	Erro        string  `json:"erro,omitempty"`
}

// ── Repository ──────────────────────────────────────────────────────────────

func (r *repository) ConsultasAtivas(ctx context.Context) ([]ConsultaAtiva, error) {
	// Abaixo de 5 segundos é ruído: o que interessa aqui é consulta travada.
	rows, err := r.pool.Query(ctx, `
		SELECT pid, state, wait_event_type,
		       EXTRACT(EPOCH FROM (now() - query_start))::float8,
		       left(query, 500), COALESCE(application_name, '')
		FROM pg_stat_activity
		WHERE state = 'active'
		  AND pid <> pg_backend_pid()
		  AND query_start IS NOT NULL
		  AND now() - query_start > interval '5 seconds'
		ORDER BY query_start
	`)
	if err != nil {
		return nil, fmt.Errorf("sync.repository.ConsultasAtivas: %w", err)
	}
	defer rows.Close()

	consultas := []ConsultaAtiva{}
	for rows.Next() {
		var c ConsultaAtiva
		if err := rows.Scan(&c.PID, &c.Estado, &c.EsperandoPor, &c.DuracaoSeg, &c.Query, &c.Aplicacao); err != nil {
			return nil, fmt.Errorf("sync.repository.ConsultasAtivas scan: %w", err)
		}
		consultas = append(consultas, c)
	}
	return consultas, rows.Err()
}

// CancelarConsulta usa pg_cancel_backend, que cancela a consulta, e não
// pg_terminate_backend, que derruba a conexão. Mais brando e suficiente para um
// REFRESH travado, que é transacional e deixa a view com o conteúdo anterior.
//
// Retorna false quando o PID já não existe — normalmente porque terminou sozinho.
func (r *repository) CancelarConsulta(ctx context.Context, pid int32) (bool, error) {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT pg_cancel_backend($1)`, pid).Scan(&ok); err != nil {
		return false, fmt.Errorf("sync.repository.CancelarConsulta: %w", err)
	}
	return ok, nil
}

func (r *repository) SchemaDoGrupo(ctx context.Context, grupoID string) (string, error) {
	var schema string
	err := r.pool.QueryRow(ctx,
		`SELECT schema_name FROM _etl.grupos WHERE id = $1 AND deleted_at IS NULL`, grupoID,
	).Scan(&schema)
	if err != nil {
		return "", fmt.Errorf("sync.repository.SchemaDoGrupo: %w", err)
	}
	return schema, nil
}

// RefreshView prefere CONCURRENTLY, que não bloqueia leitura. Cai para o modo
// bloqueante quando a view ainda não está populada — situação em que o Postgres
// recusa o concorrente — ou quando o índice único ainda não existe.
func (r *repository) RefreshView(ctx context.Context, schema, view string) (concorrente bool, err error) {
	alvo := fmt.Sprintf("%s.%s", pgx.Identifier{schema}.Sanitize(), pgx.Identifier{view}.Sanitize())

	var populada bool
	if err := r.pool.QueryRow(ctx, `
		SELECT c.relispopulated
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2
	`, schema, view).Scan(&populada); err != nil {
		return false, fmt.Errorf("sync.repository.RefreshView view não encontrada: %w", err)
	}

	if populada {
		if _, e := r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY "+alvo); e == nil {
			return true, nil
		}
	}
	if _, e := r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW "+alvo); e != nil {
		return false, fmt.Errorf("sync.repository.RefreshView: %w", e)
	}
	return false, nil
}

// ── Service ─────────────────────────────────────────────────────────────────

func (s *service) ListarConsultasAtivas(ctx context.Context) ([]ConsultaAtiva, error) {
	return s.repo.ConsultasAtivas(ctx)
}

func (s *service) CancelarConsulta(ctx context.Context, pid int32) (bool, error) {
	cancelado, err := s.repo.CancelarConsulta(ctx, pid)
	if err != nil {
		return false, err
	}
	s.log.Warn().Int32("pid", int32(pid)).Bool("cancelado", cancelado).
		Msg("sync.service: consulta cancelada manualmente por admin_global")
	return cancelado, nil
}

func (s *service) RefreshViewsGrupo(ctx context.Context, grupoID string) ([]RefreshViewResultado, error) {
	schema, err := s.repo.SchemaDoGrupo(ctx, grupoID)
	if err != nil {
		return nil, apperror.NotFound("grupo não encontrado")
	}

	views := []string{"matvw_gerencial_ano_corrente", "matvw_gerencial_historico"}
	out := make([]RefreshViewResultado, 0, len(views))
	for _, v := range views {
		inicio := time.Now()
		concorrente, err := s.repo.RefreshView(ctx, schema, v)
		res := RefreshViewResultado{View: v, Concorrente: concorrente, DuracaoSeg: time.Since(inicio).Seconds()}
		if err != nil {
			res.Erro = err.Error()
			s.log.Warn().Err(err).Str("schema", schema).Str("view", v).Msg("sync.service: refresh manual falhou")
		}
		out = append(out, res)
	}
	return out, nil
}

// ── Handler ─────────────────────────────────────────────────────────────────

func (h *Handler) AdminConsultasAtivas(w http.ResponseWriter, r *http.Request) {
	consultas, err := h.svc.ListarConsultasAtivas(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar consultas ativas", err)
		return
	}
	response.OK(w, consultas)
}

func (h *Handler) AdminCancelarConsulta(w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.Atoi(chi.URLParam(r, "pid"))
	if err != nil || pid <= 0 {
		response.FromAppError(w, apperror.Unprocessable("pid inválido"))
		return
	}

	cancelado, err := h.svc.CancelarConsulta(r.Context(), int32(pid))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao cancelar consulta", err)
		return
	}
	response.OK(w, map[string]any{"pid": pid, "cancelado": cancelado})
}

func (h *Handler) AdminRefreshViews(w http.ResponseWriter, r *http.Request) {
	resultados, err := h.svc.RefreshViewsGrupo(r.Context(), chi.URLParam(r, "grupoID"))
	if err != nil {
		response.FromAppError(w, err)
		return
	}
	response.OK(w, resultados)
}
