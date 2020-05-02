package proxy

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel"
)

// tunnelHandler handles websocket based secure tunneling sessions.
type tunnelHandler struct {
	tunnels   map[string]*tunnelInfo
	destToken map[string]*tunnelInfo
	srcToken  map[string]*tunnelInfo
	mu        sync.Mutex
}

func (h *tunnelHandler) add(ti *tunnelInfo) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := "id"
	h.tunnels[id] = ti
	h.destToken[ti.destAccessToken] = ti
	h.srcToken[ti.srcAccessToken] = ti

	return id, nil
}

func (h *tunnelHandler) remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ti, ok := h.tunnels[id]
	if !ok {
		return errors.New("tunnel not found")
	}

	delete(h.destToken, ti.destAccessToken)
	delete(h.srcToken, ti.srcAccessToken)
	delete(h.tunnels, id)

	return nil
}

func (h *tunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Header["Access-Token"]
	if !ok || len(a) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ti *tunnelInfo
	var chRead, chWrite chan []byte

	q := r.URL.Query()
	mode := tunnel.ClientMode(q.Get("local-proxy-mode"))
	switch mode {
	case tunnel.Source:
		h.mu.Lock()
		ti, ok = h.srcToken[a[0]]
		h.mu.Unlock()
		chRead = ti.chDestSrc
		chWrite = ti.chSrcDest
	case tunnel.Destination:
		h.mu.Lock()
		ti, ok = h.destToken[a[0]]
		h.mu.Unlock()
		chRead = ti.chSrcDest
		chWrite = ti.chDestSrc
	default:
		http.Error(w, "Invalid local-proxy-mode", http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s := &websocket.Server{
		Handshake: func(cfg *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			ws.PayloadType = websocket.BinaryFrame
			defer func() {
				_ = ws.Close()
			}()

			go func() {
				b := make([]byte, 8192)
				for {
					n, err := ws.Read(b)
					if err != nil {
						if err == io.EOF {
							return
						}
						log.Print(err)
						return
					}
					select {
					case <-ti.chClosed:
						return
					case chWrite <- b[:n]:
					}
				}
			}()
			for {
				select {
				case <-ti.chClosed:
					return
				case b := <-chRead:
					if _, err := ws.Write(b); err != nil {
						log.Print(err)
						return
					}
				}
			}
		}),
	}
	s.ServeHTTP(w, r)
}

func newTunnelHandler() *tunnelHandler {
	return &tunnelHandler{
		tunnels:   make(map[string]*tunnelInfo),
		destToken: make(map[string]*tunnelInfo),
		srcToken:  make(map[string]*tunnelInfo),
	}
}
