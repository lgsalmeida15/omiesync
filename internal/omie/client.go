package omie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL           = "https://app.omie.com.br/api/v1"
	defaultTimeout    = 30 * time.Second
	defaultPageSize   = 50
	maxRetries        = 3
)

var retryDelays = []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}

// Client é o cliente HTTP para a API do Omie.
type Client struct {
	appKey    string
	appSecret string
	http      *http.Client
	baseURL   string

	// LastMaskedPayload contém o JSON do último request com app_secret mascarado.
	LastMaskedPayload []byte
	// LastResponseMeta contém metadados da última resposta (sem dados de negócio).
	LastResponseMeta []byte
}

// maskSecret oculta parte do segredo para logs/auditoria.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}

// SetBaseURL substitui a URL base — usado em testes.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// NewClient cria um cliente Omie para uma empresa específica.
func NewClient(appKey, appSecret string) *Client {
	return &Client{
		appKey:    appKey,
		appSecret: appSecret,
		http:      &http.Client{Timeout: defaultTimeout},
		baseURL:   baseURL,
	}
}

// CallPublic é a versão pública de call — usada pelos executors ETL.
func (c *Client) CallPublic(ctx context.Context, path, method string, params any, out any) error {
	return c.call(ctx, path, method, params, out)
}

// call executa uma chamada à API do Omie com retry automático.
func (c *Client) call(ctx context.Context, path, method string, params any, out any) error {
	req := BaseRequest{
		AppKey:    c.appKey,
		AppSecret: c.appSecret,
		Call:      method,
		Param:     []any{params},
	}

	// Salva snapshot mascarado antes de enviar
	masked := BaseRequest{
		AppKey:    c.appKey,
		AppSecret: maskSecret(c.appSecret),
		Call:      method,
		Param:     req.Param,
	}
	// Inclui URL no payload para inspeção completa
	payloadMap := map[string]any{
		"url":        c.baseURL + path,
		"app_key":    masked.AppKey,
		"app_secret": masked.AppSecret,
		"call":       masked.Call,
		"param":      masked.Param,
	}
	c.LastMaskedPayload, _ = json.Marshal(payloadMap)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("omie.client.call marshal: %w", err)
	}

	url := c.baseURL + path
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("omie.client.call: %w", ctx.Err())
			case <-time.After(retryDelays[attempt-1]):
			}
		}

		lastErr = c.doRequest(ctx, url, body, out)
		if lastErr == nil {
			return nil
		}

		// Erros de infraestrutura Omie (BG indisponível) são retentáveis
		if omieErr, isOmieErr := lastErr.(OmieError); isOmieErr {
			if omieErr.FaultCode == "SOAP-ENV:Server" {
				continue
			}
			return lastErr
		}
	}

	return fmt.Errorf("omie.client.call após %d tentativas: %w", maxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, url string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executar request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ler response: %w", err)
	}

	// Prepara metadados da resposta
	meta := map[string]any{
		"http_status": resp.StatusCode,
	}

	// Tenta extrair paginação se possível (envelope genérico)
	var pag struct {
		Pagina           int    `json:"pagina"`
		TotalDePaginas   int    `json:"total_de_paginas"`
		Registros        int    `json:"registros"`
		TotalDeRegistros int    `json:"total_de_registros"`
		FaultString      string `json:"faultstring"`
	}
	_ = json.Unmarshal(raw, &pag)

	if pag.Pagina > 0 {
		meta["pagina"] = pag.Pagina
		meta["total_de_paginas"] = pag.TotalDePaginas
		meta["registros"] = pag.Registros
		meta["total_de_registros"] = pag.TotalDeRegistros
	}
	if pag.FaultString != "" {
		meta["omie_error"] = pag.FaultString
	}
	c.LastResponseMeta, _ = json.Marshal(meta)

	// Verifica se a resposta é um erro do Omie
	var omieErr OmieError
	if err := json.Unmarshal(raw, &omieErr); err == nil && omieErr.FaultCode != "" {
		return omieErr
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("omie HTTP %d: %s", resp.StatusCode, string(raw))
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("deserializar resposta: %w", err)
	}

	return nil
}
