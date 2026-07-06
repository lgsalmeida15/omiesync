package sync

import (
	"sync"
)

// SSEEvent representa um evento a ser enviado ao cliente.
type SSEEvent struct {
	Type string `json:"type"` // "job.iniciado", "modulo.progresso", etc.
	Data any    `json:"data"`
}

// SSEHub gerencia canais de escuta por empresa.
// Cada empresa tem um canal de broadcast; múltiplos clientes (abas)
// podem escutar a mesma empresa simultaneamente.
type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan SSEEvent // key: empresaID
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string][]chan SSEEvent),
	}
}

// Subscribe registra um canal para receber eventos de uma empresa.
// Retorna o canal e uma função de cancelamento (unsubscribe).
func (h *SSEHub) Subscribe(empresaID string) (chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 256)
	h.mu.Lock()
	h.subscribers[empresaID] = append(h.subscribers[empresaID], ch)
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subscribers[empresaID]
		for i, c := range subs {
			if c == ch {
				h.subscribers[empresaID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
		if len(h.subscribers[empresaID]) == 0 {
			delete(h.subscribers, empresaID)
		}
	}
	return ch, cancel
}

// Publish envia um evento para todos os assinantes de uma empresa.
// Non-blocking: se o canal estiver cheio, o evento é descartado.
func (h *SSEHub) Publish(empresaID string, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subscribers[empresaID] {
		select {
		case ch <- event:
		default: // canal cheio → descarta (não bloqueia o worker)
		}
	}
}
