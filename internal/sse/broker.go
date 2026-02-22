package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// Broker manages SSE client connections and broadcasts events to them.
type Broker struct {
	mu      sync.RWMutex
	clients map[string]map[chan []byte]struct{} // userID -> set of client channels
}

// NewBroker creates a new SSE broker.
func NewBroker() *Broker {
	return &Broker{
		clients: make(map[string]map[chan []byte]struct{}),
	}
}

// Subscribe registers a new SSE client for the given user and returns an event channel.
func (b *Broker) Subscribe(userID string) chan []byte {
	ch := make(chan []byte, 64)

	b.mu.Lock()
	if b.clients[userID] == nil {
		b.clients[userID] = make(map[chan []byte]struct{})
	}
	b.clients[userID][ch] = struct{}{}
	b.mu.Unlock()

	slog.Info("SSE client subscribed", "user_id", userID)
	return ch
}

// Unsubscribe removes an SSE client channel for the given user.
func (b *Broker) Unsubscribe(userID string, ch chan []byte) {
	b.mu.Lock()
	if userClients, ok := b.clients[userID]; ok {
		delete(userClients, ch)
		if len(userClients) == 0 {
			delete(b.clients, userID)
		}
	}
	b.mu.Unlock()
	close(ch)

	slog.Info("SSE client unsubscribed", "user_id", userID)
}

// Broadcast sends data to all SSE clients connected for the given user.
func (b *Broker) Broadcast(userID string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal SSE payload", "error", err)
		return
	}

	b.mu.RLock()
	userClients := b.clients[userID]
	b.mu.RUnlock()

	for ch := range userClients {
		select {
		case ch <- payload:
		default:
			slog.Warn("SSE client buffer full, dropping event", "user_id", userID)
		}
	}
}

// ServeHTTP handles an SSE connection for the given user.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := b.Subscribe(userID)
	defer b.Unsubscribe(userID, ch)

	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: alert\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
