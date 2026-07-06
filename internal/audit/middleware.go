package audit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const CtxKeyRequestID contextKey = "request_id"

// skipResponseBodyPaths lista prefixos de path cujo response_body não deve ser
// gravado em claro no audit_log — evita persistir dados financeiros/pessoais.
// O request_body ainda é gravado normalmente para fins de auditoria de ação.
var skipResponseBodyPaths = []string{
	"/query",  // SQL Explorer — resposta pode conter linhas de tabelas de clientes
	"/dados/", // endpoints de dados financeiros
}

func shouldSkipResponseBody(path string) bool {
	for _, prefix := range skipResponseBodyPaths {
		if strings.Contains(path, prefix) {
			return true
		}
	}
	return false
}

// responseRecorder captura o status code e body da response.
type responseRecorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.buf.Write(b)
	return rr.ResponseWriter.Write(b)
}

// Flush implementa http.Flusher delegando ao ResponseWriter subjacente.
// Necessário para SSE: sem isso o cast w.(http.Flusher) falha no handler de stream.
func (rr *responseRecorder) Flush() {
	if f, ok := rr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Middleware retorna um handler que grava audit_logs de forma assíncrona.
// repo pode ser nil — nesse caso apenas loga via zerolog sem persistir.
func Middleware(repo Repository, log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Lê body do request (para auditoria) e devolve ao stream
			var reqBody string
			if r.Body != nil && r.ContentLength != 0 {
				raw, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(raw))
				reqBody = SanitizeBody(string(raw))
			}

			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rr, r)

			duration := time.Since(start).Milliseconds()
			var respBody string
			if shouldSkipResponseBody(r.URL.Path) {
				respBody = "[omitido: conteúdo sensível]"
			} else {
				respBody = SanitizeBody(rr.buf.String())
			}

			requestID, _ := r.Context().Value(CtxKeyRequestID).(string)

			entry := LogEntry{
				RequestID:    requestID,
				Method:       r.Method,
				Path:         r.URL.Path,
				QueryParams:  r.URL.RawQuery,
				StatusCode:   rr.status,
				RequestBody:  reqBody,
				ResponseBody: respBody,
				IPAddress:    realIP(r),
				UserAgent:    r.UserAgent(),
				DurationMs:   duration,
			}

			// Dispara gravação assíncrona — nunca bloqueia o handler
			// Usa context.Background() pois r.Context() é cancelado ao fim do request
			if repo != nil {
				go func(e LogEntry) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := repo.Insert(ctx, e); err != nil {
						log.Error().Err(err).Msg("audit: falha ao gravar log")
					}
				}(entry)
			}
		})
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
