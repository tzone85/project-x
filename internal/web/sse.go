package web

import (
	"fmt"
	"net/http"
	"sync"
)

// SSEHub manages Server-Sent Events connections. It is safe for
// concurrent use by multiple goroutines.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewSSEHub creates an SSEHub ready to accept clients.
func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan string]struct{})}
}

// ServeHTTP upgrades an HTTP connection to an SSE stream and blocks
// until the client disconnects or the request context is cancelled.
func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		// Drain any buffered messages so they are not leaked.
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}()

	// Send initial connection confirmation.
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// Broadcast sends a JSON-formatted message to all connected SSE clients.
// Slow clients that cannot keep up have their messages dropped.
func (h *SSEHub) Broadcast(eventType, data string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	msg := fmt.Sprintf("{\"type\":%q,\"data\":%s}", eventType, data)
	for ch := range h.clients {
		select {
		case ch <- msg:
		default: // drop if client is slow
		}
	}
}

// ClientCount returns the number of currently connected SSE clients.
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
