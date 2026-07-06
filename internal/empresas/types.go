package empresas

import "time"

type Empresa struct {
	ID           string
	GrupoID      string
	Nome         string
	CNPJ         string
	AppKey       string
	AppSecret    string
	Status       string
	StatusSync   string
	UltimoSyncAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// EmpresaResponse omite app_secret conforme regra absoluta do CLAUDE.md
type EmpresaResponse struct {
	ID           string     `json:"id"`
	GrupoID      string     `json:"grupo_id"`
	Nome         string     `json:"nome"`
	CNPJ         string     `json:"cnpj,omitempty"`
	AppKey       string     `json:"app_key"`
	AppSecret    string     `json:"app_secret"` // sempre mascarado via maskSecret()
	Status       string     `json:"status"`
	StatusSync   string     `json:"status_sync"`
	UltimoSyncAt *time.Time `json:"ultimo_sync_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	Nome      string `json:"nome"`
	CNPJ      string `json:"cnpj"`
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}

type UpdateRequest struct {
	Nome      string `json:"nome"`
	CNPJ      string `json:"cnpj"`
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}

type ListParams struct {
	GrupoID string
	Page    int
	PerPage int
}

// maskSecret retorna os primeiros 4 chars + "****"
func maskSecret(s string) string {
	if len(s) < 4 {
		return "****"
	}
	return s[:4] + "****"
}

func toResponse(e *Empresa) *EmpresaResponse {
	return &EmpresaResponse{
		ID:           e.ID,
		GrupoID:      e.GrupoID,
		Nome:         e.Nome,
		CNPJ:         e.CNPJ,
		AppKey:       e.AppKey,
		AppSecret:    maskSecret(e.AppSecret),
		Status:       e.Status,
		StatusSync:   e.StatusSync,
		UltimoSyncAt: e.UltimoSyncAt,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
