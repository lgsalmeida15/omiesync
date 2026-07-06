package omie

// BaseRequest é o envelope de todas as chamadas à API do Omie.
type BaseRequest struct {
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
	Call      string `json:"call"`
	Param     []any  `json:"param"`
}

// PaginacaoParams parâmetros padrão de paginação do Omie.
type PaginacaoParams struct {
	Pagina              int    `json:"pagina"`
	RegistrosPorPagina  int    `json:"registros_por_pagina"`
	ApenasImportadoAPI  string `json:"apenas_importado_api,omitempty"`
	FiltrarPorDataDe    string `json:"filtrar_por_data_de,omitempty"`
	FiltrarPorDataAte   string `json:"filtrar_por_data_ate,omitempty"`
}

// PaginacaoResponse campos de paginação presentes em toda resposta de lista.
type PaginacaoResponse struct {
	Pagina             int `json:"pagina"`
	TotalDePaginas     int `json:"total_de_paginas"`
	Registros          int `json:"registros"`
	TotalDeRegistros   int `json:"total_de_registros"`
}
