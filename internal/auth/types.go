package auth

import "time"

type Usuario struct {
	ID        string
	GrupoID   string
	Nome      string
	Email     string
	Password  string
	Role      string
	Ativo     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RefreshToken struct {
	ID        string
	UsuarioID string
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
	// GrupoID é o grupo ativo quando a sessão foi criada. Sem ele o /auth/refresh
	// não teria como saber qual grupo o usuário multi-grupo selecionou — o refresh
	// token é opaco e não carrega claims.
	GrupoID string
}

type GrupoInfo struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Slug       string `json:"slug"`
	SchemaName string `json:"schema_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse cobre dois cenários:
// 1. Login direto (único grupo ou grupo padrão): access_token + refresh_token preenchidos.
// 2. Seleção pendente (múltiplos grupos): needs_select=true, pre_auth_token + grupos preenchidos.
type LoginResponse struct {
	// Cenário 1 — login completo
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`

	// Cenário 2 — seleção de grupo pendente
	NeedsSelect   bool        `json:"needs_select,omitempty"`
	PreAuthToken  string      `json:"pre_auth_token,omitempty"`
	Grupos        []GrupoInfo `json:"grupos,omitempty"`
}

type SelectGrupoRequest struct {
	PreAuthToken string `json:"pre_auth_token"`
	GrupoID      string `json:"grupo_id"`
}

type TrocaGrupoRequest struct {
	GrupoID string `json:"grupo_id"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type MeResponse struct {
	ID      string `json:"id"`
	GrupoID string `json:"grupo_id"`
	Nome    string `json:"nome"`
	Email   string `json:"email"`
	Role    string `json:"role"`
}
