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
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
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
