package tunnel

import (
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

func pingMarshal(v interface{}) (msg []byte, payloadType byte, err error) {
	return []byte{}, websocket.PingFrame, nil
}

func pingUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	return nil
}

func newPinger(ws *websocket.Conn, period time.Duration) func() {
	var doneOnce sync.Once
	done := make(chan struct{})

	pingMessage := websocket.Codec{Marshal: pingMarshal, Unmarshal: pingUnmarshal}
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(period):
				_ = pingMessage.Send(ws, nil)
			}
		}
	}()
	return func() {
		doneOnce.Do(func() {
			close(done)
		})
	}
}
