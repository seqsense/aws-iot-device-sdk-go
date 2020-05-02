package proxy

import (
	"errors"
	"net/http"
	"sync"
)

// tunnelHandler handles websocket based secure tunneling sessions.
type tunnelHandler struct {
	tunnels map[string]*tunnelInfo
	mu      sync.Mutex
}

func (h *tunnelHandler) add(ti *tunnelInfo) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := "id"
	h.tunnels[id] = ti

	return id, nil
}

func (h *tunnelHandler) remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.tunnels[id]; !ok {
		return errors.New("tunnel not found")
	}

	delete(h.tunnels, id)

	return nil
}

func (h *tunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func newTunnelHandler() *tunnelHandler {
	return &tunnelHandler{
		tunnels: make(map[string]*tunnelInfo),
	}
}
