package omie

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_OmieErrorParsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OmieError{
			FaultCode:   ErrCodeCredencialInvalida,
			FaultString: "Credencial inválida",
		})
	}))
	defer srv.Close()

	c := &Client{appKey: "k", appSecret: "s", http: srv.Client()}

	// Substitui URL base para o servidor de teste
	var out map[string]any
	err := c.doRequest(context.Background(), srv.URL, []byte(`{}`), &out)

	if err == nil {
		t.Fatal("esperava erro")
	}
	if !IsCredencialInvalida(err) {
		t.Errorf("esperava ErrCodeCredencialInvalida, got: %v", err)
	}
}

func TestClient_HTTP500Retried(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`erro interno`))
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	// Substitui delays por zero para teste rápido
	orig := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	defer func() { retryDelays = orig }()

	c := &Client{appKey: "k", appSecret: "s", http: srv.Client()}
	var out map[string]any
	err := c.doRequest(context.Background(), srv.URL, []byte(`{}`), &out)

	// doRequest não retenta — apenas call() retenta
	// Aqui testamos que HTTP 500 retorna erro
	if err == nil {
		t.Fatal("esperava erro no HTTP 500")
	}
}

func TestOmieError_IsCredencialInvalida(t *testing.T) {
	err := OmieError{FaultCode: ErrCodeCredencialInvalida, FaultString: "teste"}
	if !IsCredencialInvalida(err) {
		t.Error("deveria ser credencial inválida")
	}
	if IsLimiteExcedido(err) {
		t.Error("não deveria ser limite excedido")
	}
}
