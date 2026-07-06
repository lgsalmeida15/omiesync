package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
)

// privateRanges lista os blocos CIDR que não devem ser alcançáveis via webhook.
var privateRanges = func() []*net.IPNet {
	blocks := []string{
		"127.0.0.0/8",     // loopback IPv4
		"169.254.0.0/16",  // link-local / Azure IMDS
		"10.0.0.0/8",      // RFC-1918
		"172.16.0.0/12",   // RFC-1918
		"192.168.0.0/16",  // RFC-1918
		"::1/128",         // loopback IPv6
		"fc00::/7",        // ULA IPv6
	}
	var nets []*net.IPNet
	for _, b := range blocks {
		_, ipnet, _ := net.ParseCIDR(b)
		if ipnet != nil {
			nets = append(nets, ipnet)
		}
	}
	return nets
}()

// validateWebhookURL rejeita URLs que apontam para redes internas ou protocolos
// não esperados, prevenindo ataques SSRF.
func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url inválida: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme não permitido: %q (apenas http/https)", u.Scheme)
	}

	hostname := u.Hostname()
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("falha ao resolver hostname %q: %w", hostname, err)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, block := range privateRanges {
			if block.Contains(ip) {
				return fmt.Errorf("endereço %s do host %q está em range privado/reservado (%s)", ip, hostname, block)
			}
		}
	}
	return nil
}

const (
	maxRetries    = 3
	httpTimeout   = 10 * time.Second
)

var backoffDelays = []time.Duration{30 * time.Second, 2 * time.Minute, 5 * time.Minute}

type Dispatcher interface {
	Dispatch(grupoID string, event Event)
}

type dispatcher struct {
	repo   Repository
	log    zerolog.Logger
	client *http.Client
}

func NewDispatcher(repo Repository, log zerolog.Logger) Dispatcher {
	return &dispatcher{
		repo: repo,
		log:  log.With().Str("component", "webhook_dispatcher").Logger(),
		client: &http.Client{
			Timeout: httpTimeout,
			// Não seguir redirects automaticamente — evita bypass de validação SSRF
			// via redirect para endereço interno.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Dispatch dispara o evento de forma assíncrona — nunca bloqueia o handler.
func (d *dispatcher) Dispatch(grupoID string, event Event) {
	go d.dispatch(grupoID, event)
}

func (d *dispatcher) dispatch(grupoID string, event Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	webhooks, err := d.repo.ListByGrupo(ctx, grupoID)
	if err != nil {
		d.log.Error().Err(err).Str("grupo_id", grupoID).Msg("falha ao carregar webhooks")
		return
	}

	for _, wh := range webhooks {
		if !eventoAtivo(wh.Eventos, event.Tipo) {
			continue
		}
		d.sendWithRetry(wh, event)
	}
}

func (d *dispatcher) sendWithRetry(wh *Webhook, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		d.log.Error().Err(err).Msg("falha ao serializar evento")
		return
	}

	// Valida a URL antes de qualquer tentativa — erros SSRF não são retentáveis.
	if err := validateWebhookURL(wh.URL); err != nil {
		d.log.Warn().
			Err(err).
			Str("url", wh.URL).
			Str("evento", event.Tipo).
			Msg("webhook descartado: url rejeitada por segurança")
		return
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoffDelays[attempt-1])
		}

		if err := d.send(wh.URL, payload); err != nil {
			d.log.Warn().
				Err(err).
				Str("url", wh.URL).
				Str("evento", event.Tipo).
				Int("attempt", attempt+1).
				Msg("tentativa de webhook falhou")
			continue
		}

		d.log.Info().
			Str("url", wh.URL).
			Str("evento", event.Tipo).
			Msg("webhook entregue")
		return
	}

	d.log.Error().
		Str("url", wh.URL).
		Str("evento", event.Tipo).
		Msg("webhook falhou após todas as tentativas")
}

func (d *dispatcher) send(rawURL string, payload []byte) error {
	if err := validateWebhookURL(rawURL); err != nil {
		return fmt.Errorf("url rejeitada por segurança: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("enviar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status inesperado: %d", resp.StatusCode)
	}
	return nil
}

func eventoAtivo(eventos []string, tipo string) bool {
	for _, e := range eventos {
		if e == tipo {
			return true
		}
	}
	return false
}
