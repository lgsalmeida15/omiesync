package ia

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"omie-sync-api/internal/apperror"
	"omie-sync-api/internal/auth"
	"omie-sync-api/internal/response"
)

/*
orcamentoPergunta é o teto de uma pergunta inteira, da entrada ao JSON de volta.

É o número que governa todos os outros prazos do módulo: o timeout por tentativa
do cliente (client.go) e o da ferramenta (tools.go) são recortes deste. Antes
não havia um teto declarado em lugar nenhum — cada camada tinha o seu, eles não
conversavam, e a soma passava de quatro minutos contra um servidor que cortava
aos quinze segundos.

folgaEscrita é o respiro entre o fim do trabalho e o prazo de escrita da
resposta: o deadline do socket precisa sobreviver ao último Write, senão o
usuário perde justamente a resposta que deu certo.
*/
const (
	orcamentoPergunta = 90 * time.Second
	folgaEscrita      = 10 * time.Second
)

type Handler struct {
	svc     Service
	jwtSvc  auth.JWTService
	membros auth.MembroChecker
	log     zerolog.Logger
	// limite é criado UMA vez no wire. Instanciar por request criaria um
	// limitador novo a cada chamada, e nada seria limitado.
	limite func(http.Handler) http.Handler
}

func NewHandler(svc Service, jwtSvc auth.JWTService, membros auth.MembroChecker, log zerolog.Logger, limite func(http.Handler) http.Handler) *Handler {
	return &Handler{svc: svc, jwtSvc: jwtSvc, membros: membros, log: log, limite: limite}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(h.jwtSvc))
	// Sem RequireRole: admin de grupo e viewer usam o chat. Ele não revela nada
	// além do que a pessoa já vê no dashboard — as ferramentas rodam o mesmo
	// código, com o mesmo isolamento.
	r.Use(auth.RequireGrupoMembro(h.membros))

	r.Get("/disponivel", h.Disponivel)
	r.Get("/conversa", h.Historico)
	r.Delete("/conversa", h.Limpar)
	// Só a pergunta é limitada: é a única que custa dinheiro.
	r.With(h.limite).Post("/chat", h.Chat)

	return r
}

/*
grupoDoContexto: o grupo vem SEMPRE das claims.

Não há parâmetro de rota nem de corpo para escolher grupo. É a mesma garantia
das ferramentas, um nível acima.

Contexto de plataforma não tem grupo — e o assistente é sobre dados de um
cliente, então ali ele não existe.
*/
func grupoDoContexto(r *http.Request) (claims *auth.JWTClaims, grupoID string, ok bool) {
	c, existe := auth.ClaimsFromContext(r.Context())
	if !existe || c.GrupoID == "" {
		return nil, "", false
	}
	return c, c.GrupoID, true
}

// GET /ia/disponivel
func (h *Handler) Disponivel(w http.ResponseWriter, r *http.Request) {
	_, grupoID, ok := grupoDoContexto(r)
	if !ok {
		// Sem grupo não há assistente, mas também não é erro: a tela só não
		// mostra o botão.
		response.OK(w, map[string]bool{"disponivel": false})
		return
	}

	disponivel, err := h.svc.Disponivel(r.Context(), grupoID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao verificar disponibilidade", err)
		return
	}
	response.OK(w, map[string]bool{"disponivel": disponivel})
}

// POST /ia/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.Forbidden(w, "o assistente só funciona dentro de um grupo")
		return
	}

	var req PerguntaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "body inválido", err)
		return
	}

	/*
	 * Esta rota precisa de mais tempo que as outras.
	 *
	 * O WriteTimeout global é 15s (cmd/api/main.go) e protege os endpoints
	 * normais. Uma pergunta ao assistente é outra coisa: consulta ao banco, ida
	 * ao modelo, volta, e às vezes uma segunda rodada. Com 15s o servidor
	 * cortava a conexão com o corpo pela metade — o navegador não recebia JSON
	 * nenhum e mostrava um erro genérico sem causa. Era a origem mais comum do
	 * "não consegui responder agora".
	 *
	 * SetWriteDeadline estende o prazo APENAS aqui. Se o servidor não suportar
	 * (ResponseWriter embrulhado por algum middleware), seguimos assim mesmo: o
	 * contexto ainda limita o trabalho, e falhar a pergunta por causa do prazo
	 * seria pior que o comportamento de antes.
	 */
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(orcamentoPergunta + folgaEscrita)); err != nil {
		h.log.Debug().Err(err).Msg("ia: não foi possível estender o prazo de escrita")
	}

	ctx, cancel := context.WithTimeout(r.Context(), orcamentoPergunta)
	defer cancel()

	resp, err := h.svc.Perguntar(ctx, grupoID, claims.UserID, req)
	if err != nil {
		if ae, ok := apperror.IsAppError(err); ok {
			response.FromAppError(w, ae)
			return
		}

		/*
		 * O detalhe do erro fica no log, não na resposta.
		 *
		 * response.Error anexa err.Error() ao corpo JSON, e aqui esse erro pode
		 * carregar a URL do provedor ou parte do prompt — e o prompt tem dado
		 * financeiro de cliente. O request_id liga as duas pontas para quem for
		 * investigar.
		 */
		h.log.Error().Err(err).Str("grupo_id", grupoID).Msg("ia: falha ao responder")
		response.Error(w, statusDaFalha(err), mensagemDaFalha(err), nil)
		return
	}
	response.OK(w, resp)
}

/*
mensagemDaFalha traduz a causa para algo que a pessoa possa agir a respeito.

Antes tudo virava "o assistente não conseguiu responder agora" — credencial
errada, provedor sobrecarregado e demora liam igual, e nenhuma das três dizia o
que fazer. Quem vê a mensagem nem sempre é quem configura, então o texto precisa
funcionar para os dois: o usuário entende que não é erro dele, e o administrador
reconhece o que ajustar.
*/
func mensagemDaFalha(err error) string {
	switch {
	case errors.Is(err, ErrCredencialRecusada):
		return "A credencial da IA foi recusada pelo provedor. Avise o administrador."
	case errors.Is(err, ErrSemCredencial):
		return "O assistente ainda não está configurado."
	case errors.Is(err, ErrProvedorOcupado):
		return "O provedor está limitando as consultas. Tente de novo em alguns instantes."
	case errors.Is(err, context.DeadlineExceeded):
		return "A consulta demorou mais que o esperado. Tente uma pergunta mais específica."
	}
	return "o assistente não conseguiu responder agora"
}

// statusDaFalha: o que depende de configuração ou do provedor não é erro do
// servidor. 503 também diz ao navegador que tentar de novo pode funcionar.
func statusDaFalha(err error) int {
	switch {
	case errors.Is(err, ErrCredencialRecusada), errors.Is(err, ErrSemCredencial),
		errors.Is(err, ErrProvedorOcupado), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

// GET /ia/conversa
func (h *Handler) Historico(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.OK(w, []Mensagem{})
		return
	}

	msgs, err := h.svc.Historico(r.Context(), grupoID, claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao carregar a conversa", err)
		return
	}
	response.OK(w, msgs)
}

// DELETE /ia/conversa
func (h *Handler) Limpar(w http.ResponseWriter, r *http.Request) {
	claims, grupoID, ok := grupoDoContexto(r)
	if !ok {
		response.NoContent(w)
		return
	}

	if err := h.svc.Limpar(r.Context(), grupoID, claims.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, "erro ao limpar a conversa", err)
		return
	}
	response.NoContent(w)
}
