package webhooks

import "time"

type Webhook struct {
	ID      string
	GrupoID string
	URL     string
	Secret  string
	Eventos []string
	Ativo   bool
}

type Event struct {
	Tipo      string    `json:"tipo"`
	GrupoID   string    `json:"grupo_id"`
	EmpresaID string    `json:"empresa_id,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	OcorridoAt time.Time `json:"ocorrido_at"`
}

const (
	EventEmpresaPausada   = "empresa.pausada"
	EventEmpresaReativada = "empresa.reativada"
	EventSyncFalhou       = "sync.falhou"
	EventSyncConcluido    = "sync.concluido"
)
