package sync

import "time"

type SyncJob struct {
	ID          string     `json:"id"`
	EmpresaID   string     `json:"empresa_id"`
	Tipo        string     `json:"tipo"`
	Status      string     `json:"status"`
	Erro        string     `json:"erro"`
	IniciadoAt  *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt *time.Time `json:"concluido_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Executor    string     `json:"executor,omitempty"`
}

type SyncControl struct {
	ID                     string     `json:"id"`
	EmpresaID              string     `json:"empresa_id"`
	Ativo                  bool       `json:"ativo"`
	IntervaloIncrementalMin int        `json:"intervalo_incremental_min"`
	IntervaloFullDias      int        `json:"intervalo_full_dias"`
	UltimoSyncAt           *time.Time `json:"ultimo_sync_at,omitempty"`
	ProximoSyncAt          *time.Time `json:"proximo_sync_at,omitempty"`
	UltimoFullSyncAt       *time.Time `json:"ultimo_full_sync_at,omitempty"`
	ProximoFullSyncAt      *time.Time `json:"proximo_full_sync_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type StatusResponse struct {
	EmpresaID string       `json:"empresa_id"`
	Control   *SyncControl `json:"controle"`
	UltimoJob *SyncJob     `json:"ultimo_job,omitempty"`
}

type ForcarSyncRequest struct {
	Tipo     string `json:"tipo"`
	Executor string `json:"executor,omitempty"`
}

type ConfigurarRequest struct {
	Ativo                  bool `json:"ativo"`
	IntervaloIncrementalMin int  `json:"intervalo_incremental_min"`
	IntervaloFullDias      int  `json:"intervalo_full_dias"`
}

type EmpresaExecutorConfig struct {
	Executor  string    `json:"executor"`
	Ativo     bool      `json:"ativo"`
	Notas     *string   `json:"notas,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy *string   `json:"updated_by,omitempty"`
}

type UpdateExecutorConfigRequest struct {
	Ativo bool    `json:"ativo"`
	Notas *string `json:"notas,omitempty"`
}

type SyncJobProgress struct {
	Executor       string     `json:"executor"`
	Status         string     `json:"status"`
	PaginaAtual    *int       `json:"pagina_atual,omitempty"`
	TotalPaginas   *int       `json:"total_paginas,omitempty"`
	RegistrosProc  int        `json:"registros_proc"`
	RegistrosTotal *int       `json:"registros_total,omitempty"`
	Erro           *string    `json:"erro,omitempty"`
	IniciadoAt     *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt    *time.Time `json:"concluido_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Novos campos para inspeção de payload
	UltimoPayload  any     `json:"ultimo_payload,omitempty"`
	UltimoResponse any     `json:"ultimo_response,omitempty"`
	ErroPayload    any     `json:"erro_payload,omitempty"`
	ErroResponse   *string `json:"erro_response,omitempty"`
}

type ListParams struct {
	EmpresaID string
	Page      int
	PerPage   int
}

type JobStatusCount struct {
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

type JobAtivoRow struct {
	ID                string     `json:"id"`
	EmpresaID         string     `json:"empresa_id"`
	EmpresaNome       string     `json:"empresa_nome"`
	GrupoNome         string     `json:"grupo_nome"`
	Tipo              string     `json:"tipo"`
	Status            string     `json:"status"`
	IniciadoAt        time.Time  `json:"iniciado_at"`
	UltimoHeartbeatAt *time.Time `json:"ultimo_heartbeat_at,omitempty"`
	IsZumbi           bool       `json:"is_zumbi"`
}

type JobPage struct {
	ID             string     `json:"id"`
	JobID          string     `json:"job_id"`
	Modulo         string     `json:"modulo"`
	Pagina         int        `json:"pagina"`
	TotalPaginas   int        `json:"total_paginas"`
	Tentativas     int        `json:"tentativas"`
	MaxTentativas  int        `json:"max_tentativas"`
	ProximoRetryAt *time.Time `json:"proximo_retry_at,omitempty"`
}

type DLQPageRow struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	EmpresaNome  string     `json:"empresa_nome"`
	GrupoNome    string     `json:"grupo_nome"`
	Modulo       string     `json:"modulo"`
	Pagina       int        `json:"pagina"`
	TotalPaginas int        `json:"total_paginas"`
	Tentativas   int        `json:"tentativas"`
	MaxTentativas int       `json:"max_tentativas"`
	Erro         *string    `json:"erro,omitempty"`
	ConcluidoAt  *time.Time `json:"concluido_at,omitempty"`
}

type PageRow struct {
	ID                string     `json:"id"`
	Modulo            string     `json:"modulo"`
	Pagina            int        `json:"pagina"`
	TotalPaginas      int        `json:"total_paginas"`
	Status            string     `json:"status"`
	Tentativas        int        `json:"tentativas"`
	MaxTentativas     int        `json:"max_tentativas"`
	RegistrosGravados int        `json:"registros_gravados"`
	Erro              *string    `json:"erro,omitempty"`
	ProximoRetryAt    *time.Time `json:"proximo_retry_at,omitempty"`
	IniciadoAt        *time.Time `json:"iniciado_at,omitempty"`
	ConcluidoAt       *time.Time `json:"concluido_at,omitempty"`
}
