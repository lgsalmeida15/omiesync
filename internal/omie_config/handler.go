package omie_config

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
	r.Use(auth.RequireRole("admin_global"))

	r.Get("/", h.GetAll)
	r.Get("/{modulo}", h.GetByModulo)
	r.Put("/{modulo}", h.Update)

	return r
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.svc.GetAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao listar configurações", err)
		return
	}
	response.OK(w, configs)
}

func (h *Handler) GetByModulo(w http.ResponseWriter, r *http.Request) {
	modulo := chi.URLParam(r, "modulo")
	config, err := h.svc.GetByModulo(r.Context(), modulo)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar configuração", err)
		return
	}
	response.OK(w, config)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	modulo := chi.URLParam(r, "modulo")
	claims, _ := auth.ClaimsFromContext(r.Context())
	
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	config, err := h.svc.Update(r.Context(), modulo, req, claims.UserID)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao atualizar configuração", err)
		return
	}

	response.OK(w, config)
}
