package permissoes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc auth.JWTService
}

func NewHandler(svc Service, jwtSvc auth.JWTService) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))
	r.Use(auth.RequireRole("admin_global", "admin_grupo"))

	r.Post("/grant", h.Grant)
	r.Post("/revoke", h.Revoke)
	r.Get("/usuario/{usuarioID}", h.ListByUsuario)
	r.Get("/empresa/{empresaID}", h.ListByEmpresa)

	return r
}

// POST /admin/permissoes/grant
func (h *Handler) Grant(w http.ResponseWriter, r *http.Request) {
	var req GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	p, err := h.svc.Grant(r.Context(), req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao conceder permissão", err)
		return
	}
	response.Created(w, p)
}

// POST /admin/permissoes/revoke
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	if err := h.svc.Revoke(r.Context(), req); err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao revogar permissão", err)
		return
	}
	response.NoContent(w)
}

// GET /admin/permissoes/usuario/{usuarioID}
func (h *Handler) ListByUsuario(w http.ResponseWriter, r *http.Request) {
	usuarioID := chi.URLParam(r, "usuarioID")

	ps, err := h.svc.ListByUsuario(r.Context(), usuarioID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar permissões", err)
		return
	}
	response.OK(w, ps)
}

// GET /admin/permissoes/empresa/{empresaID}
func (h *Handler) ListByEmpresa(w http.ResponseWriter, r *http.Request) {
	empresaID := chi.URLParam(r, "empresaID")

	ps, err := h.svc.ListByEmpresa(r.Context(), empresaID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar permissões", err)
		return
	}
	response.OK(w, ps)
}
