package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type mockRepo struct {
	webhooks []*Webhook
}

func (m *mockRepo) ListByGrupo(_ context.Context, _ string) ([]*Webhook, error) {
	return m.webhooks, nil
}

func newTestDispatcher(repo Repository) *dispatcher {
	return &dispatcher{
		repo:   repo,
		log:    zerolog.Nop(),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestDispatcher_DeliverEvent(t *testing.T) {
	var received atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var e Event
		_ = json.NewDecoder(r.Body).Decode(&e)
		if e.Tipo != EventSyncConcluido {
			t.Errorf("tipo: got %q want %q", e.Tipo, EventSyncConcluido)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	repo := &mockRepo{webhooks: []*Webhook{
		{ID: "w1", GrupoID: "g1", URL: srv.URL, Eventos: []string{EventSyncConcluido}},
	}}
	d := newTestDispatcher(repo)

	event := Event{Tipo: EventSyncConcluido, GrupoID: "g1", OcorridoAt: time.Now()}
	d.dispatch("g1", event)

	// Aguarda entrega (dispatch é síncrono no teste via dispatch() direto)
	time.Sleep(100 * time.Millisecond)
	if received.Load() != 1 {
		t.Errorf("received: got %d want 1", received.Load())
	}
}

func TestDispatcher_FiltraEventoInativo(t *testing.T) {
	var received atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// webhook só aceita sync.concluido, mas dispararemos empresa.pausada
	repo := &mockRepo{webhooks: []*Webhook{
		{ID: "w1", URL: srv.URL, Eventos: []string{EventSyncConcluido}},
	}}
	d := newTestDispatcher(repo)

	d.dispatch("g1", Event{Tipo: EventEmpresaPausada, OcorridoAt: time.Now()})
	time.Sleep(100 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("não deveria ter enviado: got %d", received.Load())
	}
}

func TestDispatcher_SemWebhooks(t *testing.T) {
	// Não deve panicar com lista vazia
	d := newTestDispatcher(&mockRepo{webhooks: []*Webhook{}})
	d.dispatch("g1", Event{Tipo: EventSyncFalhou, OcorridoAt: time.Now()})
	time.Sleep(50 * time.Millisecond)
}

func TestDispatcher_RetryOnFailure(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	repo := &mockRepo{webhooks: []*Webhook{
		{ID: "w1", URL: srv.URL, Eventos: []string{EventSyncFalhou}},
	}}

	// Substituir backoff por zeros para o teste não demorar
	origBackoff := backoffDelays
	backoffDelays = []time.Duration{0, 0, 0}
	defer func() { backoffDelays = origBackoff }()

	d := newTestDispatcher(repo)
	d.dispatch("g1", Event{Tipo: EventSyncFalhou, OcorridoAt: time.Now()})
	time.Sleep(200 * time.Millisecond)

	if attempts.Load() < 3 {
		t.Errorf("esperava pelo menos 3 tentativas, got %d", attempts.Load())
	}
}

func TestEventoAtivo(t *testing.T) {
	casos := []struct {
		eventos []string
		tipo    string
		want    bool
	}{
		{[]string{EventSyncConcluido, EventSyncFalhou}, EventSyncConcluido, true},
		{[]string{EventSyncConcluido}, EventEmpresaPausada, false},
		{[]string{}, EventSyncConcluido, false},
	}
	for _, tc := range casos {
		got := eventoAtivo(tc.eventos, tc.tipo)
		if got != tc.want {
			t.Errorf("eventoAtivo(%v, %q) = %v, want %v", tc.eventos, tc.tipo, got, tc.want)
		}
	}
}
