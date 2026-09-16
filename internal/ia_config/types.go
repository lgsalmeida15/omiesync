package ia_config

import "time"

/*
Config é a configuração do assistente: como falar com o provedor.

Uma só, da plataforma inteira. A credencial é da OTM e a conta é da OTM, então
não há uma por grupo — o que é por grupo é apenas o liga/desliga, em GrupoIA.
*/
type Config struct {
	Provedor string `json:"provedor"`
	Modelo   string `json:"modelo"`
	// BaseURL vazio usa o padrão do provedor. Existe para permitir trocar por
	// outro provedor compatível com OpenAI sem tocar em código.
	BaseURL string `json:"base_url"`
	// APIKey SEMPRE sai mascarada da API — ver Response.
	APIKey        string `json:"-"`
	MaxTokens     int32  `json:"max_tokens"`
	TetoTokensDia int32  `json:"teto_tokens_dia"`
	Ativo         bool   `json:"ativo"`
	// SystemPrompt vazio usa o texto versionado em internal/ia/prompt.go.
	// Preenchido, substitui o padrão por inteiro.
	SystemPrompt string `json:"system_prompt"`

	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedByEmail string    `json:"updated_by_email,omitempty"`
}

/*
Response é o que a API devolve.

Tipo separado de Config por um motivo só: garantir que a chave não escape. Com
um tipo único bastaria alguém acrescentar uma tag JSON no campo para vazar a
credencial num endpoint que admin_grupo alcança. Aqui o campo nem existe.
*/
type Response struct {
	Provedor string `json:"provedor"`
	Modelo   string `json:"modelo"`
	BaseURL  string `json:"base_url"`
	// APIKeyMascarada mostra que existe chave sem mostrar a chave.
	APIKeyMascarada string `json:"api_key_mascarada"`
	// TemChave distingue "nunca configurada" de "configurada" na tela, sem que
	// ela precise interpretar a máscara.
	TemChave bool `json:"tem_chave"`

	MaxTokens     int32 `json:"max_tokens"`
	TetoTokensDia int32 `json:"teto_tokens_dia"`
	Ativo         bool  `json:"ativo"`

	// SystemPrompt é o texto em vigor (vazio = o padrão está em uso).
	SystemPrompt string `json:"system_prompt"`
	// SystemPromptPadrao acompanha a resposta para a tela poder oferecer
	// "restaurar padrão" mostrando o que vai ser recuperado. Sem isto o
	// administrador teria de confiar num botão que não mostra o que faz.
	SystemPromptPadrao string `json:"system_prompt_padrao"`

	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedByEmail string    `json:"updated_by_email,omitempty"`
}

/*
UpdateRequest: APIKey vazia PRESERVA a chave atual.

Sem isso a tela teria de devolver a credencial para poder salvar qualquer outro
campo — e uma tela que recebe a chave em claro é uma chave a mais circulando. Com
esta regra, quem quer só mudar o modelo manda o campo vazio.

Para apagar a chave existe LimparChave, explícito, porque "apagar" e "não mexer"
não podem ser o mesmo gesto.
*/
type UpdateRequest struct {
	Provedor      string `json:"provedor"`
	Modelo        string `json:"modelo"`
	BaseURL       string `json:"base_url"`
	APIKey        string `json:"api_key"`
	LimparChave   bool   `json:"limpar_chave"`
	MaxTokens     int32  `json:"max_tokens"`
	TetoTokensDia int32  `json:"teto_tokens_dia"`
	Ativo         bool   `json:"ativo"`
	SystemPrompt  string `json:"system_prompt"`
}

// GrupoIA é a linha da tela de liga/desliga: um grupo e o estado do recurso.
type GrupoIA struct {
	GrupoID   string `json:"grupo_id"`
	GrupoNome string `json:"grupo_nome"`
	GrupoSlug string `json:"grupo_slug"`
	Ativa     bool   `json:"ativa"`
	// Zero quando ninguém nunca decidiu nada sobre este grupo.
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	UpdatedByEmail string    `json:"updated_by_email,omitempty"`
}

type SetGrupoRequest struct {
	Ativa bool `json:"ativa"`
}

/*
mascarar mostra que há chave sem mostrar a chave.

Mesmo comportamento de internal/empresas.maskSecret: chave curta demais para
revelar 4 caracteres sem entregar o resto vira só asteriscos.
*/
func mascarar(s string) string {
	if s == "" {
		return ""
	}
	if len(s) < 8 {
		return "****"
	}
	return s[:4] + "****"
}

func toResponse(c *Config) Response {
	return Response{
		Provedor:        c.Provedor,
		Modelo:          c.Modelo,
		BaseURL:         c.BaseURL,
		APIKeyMascarada: mascarar(c.APIKey),
		TemChave:        c.APIKey != "",
		MaxTokens:       c.MaxTokens,
		TetoTokensDia:   c.TetoTokensDia,
		Ativo:           c.Ativo,
		SystemPrompt:    c.SystemPrompt,
		UpdatedAt:       c.UpdatedAt,
		UpdatedByEmail:  c.UpdatedByEmail,
	}
}
