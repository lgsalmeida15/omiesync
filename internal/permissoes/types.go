package permissoes

import "time"

type Permissao struct {
	ID        string
	UsuarioID string
	EmpresaID string
	Recurso   string
	Acao      string
	CreatedAt time.Time
}

type GrantRequest struct {
	UsuarioID string `json:"usuario_id"`
	EmpresaID string `json:"empresa_id"`
	Recurso   string `json:"recurso"`
	Acao      string `json:"acao"`
}

type RevokeRequest struct {
	UsuarioID string `json:"usuario_id"`
	EmpresaID string `json:"empresa_id"`
	Recurso   string `json:"recurso"`
	Acao      string `json:"acao"`
}

var recursosValidos = map[string]bool{"dashboard": true, "sync": true, "admin": true}
var acoesValidas = map[string]bool{"ver": true, "editar": true, "forcar_sync": true}
