package server

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

// TunnelHandler handles websocket based secure tunneling sessions.
type TunnelHandler struct {
	tunnels   map[string]*tunnelInfo
	destToken map[string]*tunnelInfo
	srcToken  map[string]*tunnelInfo
	mu        sync.Mutex
	id        uint32
}

func (h *TunnelHandler) add(ti *tunnelInfo) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := fmt.Sprintf("%08x", h.id)
	h.tunnels[id] = ti
	h.destToken[ti.destAccessToken] = ti
	h.srcToken[ti.srcAccessToken] = ti

	return id, nil
}

func (h *TunnelHandler) remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ti, ok := h.tunnels[id]
	if !ok {
		return errors.New("tunnel not found")
	}

	ti.cancel()
	delete(h.destToken, ti.destAccessToken)
	delete(h.srcToken, ti.srcAccessToken)
	delete(h.tunnels, id)

	return nil
}

func (h *TunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		if ok {
			chRead = ti.chDestSrc
			chWrite = ti.chSrcDest
		}
	case tunnel.Destination:
		h.mu.Lock()
		ti, ok = h.destToken[a[0]]
		h.mu.Unlock()
		if ok {
			chRead = ti.chSrcDest
			chWrite = ti.chDestSrc
		}
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
				for {
					b := make([]byte, 8192)
					if _, err := io.ReadFull(ws, b[:2]); err != nil {
						if err == io.EOF {
							return
						}
						log.Print(err)
						return
					}
					l := int(b[0])<<8 | int(b[1])
					if cap(b) < l+2 {
						b = make([]byte, l+2)
						b[0], b[1] = byte(l>>8), byte(l)
					}
					b = b[:l+2]
					if _, err := io.ReadFull(ws, b[2:]); err != nil {
						if err == io.EOF {
							return
						}
						log.Print(err)
						return
					}
					select {
					case <-ti.chDone:
						return
					case chWrite <- b:
					}
				}
			}()
			for {
				select {
				case <-ti.chDone:
					return
				case b := <-chRead:
					for p := 0; p < len(b); {
						n, err := ws.Write(b[p:])
						if err != nil {
							log.Print(err)
							return
						}
						p += n
					}
				}
			}
		}),
	}
	s.ServeHTTP(w, r)
}

func NewTunnelHandler() *TunnelHandler {
	return &TunnelHandler{
		tunnels:   make(map[string]*tunnelInfo),
		destToken: make(map[string]*tunnelInfo),
		srcToken:  make(map[string]*tunnelInfo),
	}
}
