package sync

import (
	"testing"
	"time"
)

func TestSSEHub_PublishEntregaAoSubscriber(t *testing.T) {
	hub := NewSSEHub()
	empID := "emp-1"
	ch, cancel := hub.Subscribe(empID)
	defer cancel()

	expected := SSEEvent{Type: "test", Data: "hello"}
	hub.Publish(empID, expected)

	select {
	case evt := <-ch:
		if evt.Type != expected.Type || evt.Data != expected.Data {
			t.Errorf("evento incorreto: got %v want %v", evt, expected)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout aguardando evento")
	}
}

func TestSSEHub_CancelRemoveSubscriber(t *testing.T) {
	hub := NewSSEHub()
	empID := "emp-1"
	_, cancel := hub.Subscribe(empID)

	hub.mu.RLock()
	if len(hub.subscribers[empID]) != 1 {
		t.Errorf("deveria ter 1 subscriber, got %d", len(hub.subscribers[empID]))
	}
	hub.mu.RUnlock()

	cancel()

	hub.mu.RLock()
	if len(hub.subscribers[empID]) != 0 {
		t.Errorf("deveria ter 0 subscribers, got %d", len(hub.subscribers[empID]))
	}
	hub.mu.RUnlock()
}

func TestSSEHub_PublishSemSubscribers_NaoBloquia(t *testing.T) {
	hub := NewSSEHub()
	// Não deve travar
	hub.Publish("emp-none", SSEEvent{Type: "test", Data: "ignore"})
}

func TestSSEHub_CanalCheio_Descarta(t *testing.T) {
	hub := NewSSEHub()
	empID := "emp-1"
	ch, cancel := hub.Subscribe(empID)
	defer cancel()

	// Enche o canal (buffer de 32)
	for i := 0; i < 35; i++ {
		hub.Publish(empID, SSEEvent{Type: "test", Data: i})
	}

	// Verifica se o primeiro evento ainda está lá
	select {
	case evt := <-ch:
		if evt.Data != 0 {
			t.Errorf("esperava primeiro evento (0), got %v", evt.Data)
		}
	default:
		t.Error("canal deveria ter dados")
	}
}
