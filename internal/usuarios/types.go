package usuarios

import "time"

type Usuario struct {
	ID        string    `json:"id"`
	GrupoID   string    `json:"grupo_id"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateRequest struct {
	Nome  string `json:"nome"`
	Role  string `json:"role"`
	Ativo bool   `json:"ativo"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

type ListParams struct {
	GrupoID string
	Page    int
	PerPage int
}

var rolesValidas = map[string]bool{
	"admin_global": true,
	"admin_grupo":  true,
	"viewer":       true,
}
