package service

import (
	"sync"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Hub registers clients and broadcasts pre-marshaled payloads to those whose
// subscription matches. Broadcast is safe for concurrent use and never blocks:
// a client whose send buffer is full is treated as slow and unregistered.
type Hub struct {
	logger  log.Logger
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// NewHub builds an empty hub.
func NewHub(logger log.Logger) *Hub {
	return &Hub{logger: logger, clients: make(map[*Client]struct{})}
}

// Register starts delivering broadcasts to c. Callers must not register the
// same client twice; Unregister is the idempotent half of the pair.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

// Unregister stops delivering to c and releases its writer. It is safe to
// call more than once for the same client, including concurrently.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	c.release()
}

// Close unregisters every remaining client. It is the final step of shutdown,
// after the poll loop has stopped and the HTTP server has drained.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		delete(h.clients, c)
		c.release()
	}
}

// Broadcast delivers one payload to every registered client that wants it.
// payload must already be JSON-marshaled: the caller pays the encoding cost
// once per record instead of once per receiving connection.
func (h *Hub) Broadcast(channel Channel, symbol string, payload []byte) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if !c.wants(channel, symbol) {
			continue
		}
		c.Deliver(payload)
	}
}
