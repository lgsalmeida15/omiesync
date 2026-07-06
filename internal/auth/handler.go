package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/response"
)

type Handler struct {
	svc    Service
	jwtSvc JWTService
}

func NewHandler(svc Service, jwtSvc JWTService) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc}
}

func (h *Handler) JWTService() JWTService {
	return h.jwtSvc
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.With(httprate.LimitByIP(10, 1*time.Minute)).Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.With(httprate.LimitByIP(20, 1*time.Minute)).Post("/refresh", h.Refresh)
	r.With(RequireAuth(h.jwtSvc)).Get("/me", h.Me)

	return r
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusUnprocessableEntity, "email e password são obrigatórios", nil)
		return
	}

	resp, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro interno", err)
		return
	}

	response.OK(w, resp)
}

// POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusUnprocessableEntity, "refresh_token é obrigatório", nil)
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao revogar token", err)
		return
	}

	response.NoContent(w)
}

// POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusUnprocessableEntity, "refresh_token é obrigatório", nil)
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}
		response.Error(w, http.StatusInternalServerError, "erro ao renovar token", err)
		return
	}

	response.OK(w, resp)
}

// GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "não autenticado")
		return
	}

	me, err := h.svc.Me(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao buscar usuário", err)
		return
	}

	response.OK(w, me)
}
