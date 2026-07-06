package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"omie-sync-api/internal/response"
)

type ctxKey string

const CtxKeyUserClaims ctxKey = "user_claims"

// RequireAuth valida o Bearer token e injeta as claims no contexto.
// Aceita apenas o header Authorization: Bearer <token>.
func RequireAuth(jwtSvc JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				response.Unauthorized(w, "token não fornecido")
				return
			}

			claims, err := jwtSvc.Validate(token)
			if err != nil {
				response.Unauthorized(w, "token inválido ou expirado")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthSSE valida o Bearer token aceitando também o query param ?token=.
// Deve ser usado APENAS nas rotas SSE (Server-Sent Events), onde o browser
// não permite enviar headers customizados via EventSource.
func RequireAuthSSE(jwtSvc JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				response.Unauthorized(w, "token não fornecido")
				return
			}

			claims, err := jwtSvc.Validate(token)
			if err != nil {
				response.Unauthorized(w, "token inválido ou expirado")
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole garante que o usuário autenticado possui uma das roles permitidas.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "não autenticado")
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				response.Forbidden(w, "acesso negado")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireGrupoMembro garante que o usuário pertence ao grupo da rota.
// Extrai grupo_id do path param "grupoID".
func RequireGrupoMembro(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			response.Unauthorized(w, "não autenticado")
			return
		}

		// admin_global tem acesso irrestrito
		if claims.Role == "admin_global" {
			next.ServeHTTP(w, r)
			return
		}

		grupoID := chi.URLParam(r, "grupoID")
		if grupoID == "" {
			// sem restrição de grupo no path
			next.ServeHTTP(w, r)
			return
		}

		if claims.GrupoID != grupoID {
			response.Forbidden(w, "acesso negado a este grupo")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ClaimsFromContext extrai as claims do contexto.
func ClaimsFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(CtxKeyUserClaims).(*JWTClaims)
	return claims, ok
}

func extractBearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}
