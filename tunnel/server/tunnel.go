// Copyright 2020 SEQSENSE, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel"
	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel/msg"
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
	h.id++

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

// Clean releases expired tunnel info objects.
func (h *TunnelHandler) Clean() {
	h.mu.Lock()
	removed := make([]string, 0, len(h.tunnels))
	for id, tunnel := range h.tunnels {
		select {
		case <-tunnel.chDone:
			removed = append(removed, id)
		default:
		}
	}
	h.mu.Unlock()

	for _, id := range removed {
		h.remove(id)
	}
}

// ServeHTTP implements http.Handler.
func (h *TunnelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Header["Access-Token"]
	if !ok || len(a) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ti *tunnelInfo
	var chRead, chWrite chan []byte
	var chDone <-chan struct{}

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
			chDone = ti.chDone
		}
	case tunnel.Destination:
		h.mu.Lock()
		ti, ok = h.destToken[a[0]]
		h.mu.Unlock()
		if ok {
			chRead = ti.chSrcDest
			chWrite = ti.chDestSrc
			chDone = ti.chDone
		}
	default:
		http.Error(w, "Invalid local-proxy-mode", http.StatusBadRequest)
		return
	}
	select {
	case <-chDone:
		ok = false
	default:
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
				_ = msg.WriteMessage(ws, &msg.Message{Type: msg.Message_STREAM_RESET})
				_ = ws.Close()
			}()

			chWsClosed := make(chan struct{})
			go func() {
				defer func() {
					close(chWsClosed)
				}()
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
				case <-chWsClosed:
					return
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

// NewTunnelHandler creates tunnel WebSocket handler.
func NewTunnelHandler() *TunnelHandler {
	return &TunnelHandler{
		tunnels:   make(map[string]*tunnelInfo),
		destToken: make(map[string]*tunnelInfo),
		srcToken:  make(map[string]*tunnelInfo),
	}
}
