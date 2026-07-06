package omie_config

import "time"

type EndpointConfig struct {
	ID             string    `json:"id"`
	Modulo         string    `json:"modulo"`
	EndpointPath   string    `json:"endpoint_path"`
	Action         string    `json:"action"`
	ArrayField     string    `json:"array_field"`
	PageSize       int       `json:"page_size"`
	Ativo          bool      `json:"ativo"`
	IgnorarDelta   bool      `json:"ignorar_delta"`
	Notas          string    `json:"notas"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
	UpdatedByEmail string    `json:"updated_by_email"`
}

type UpdateRequest struct {
	EndpointPath string `json:"endpoint_path"`
	Action       string `json:"action"`
	ArrayField   string `json:"array_field"`
	PageSize     int    `json:"page_size"`
	Ativo        bool   `json:"ativo"`
	Notas        string `json:"notas"`
}
