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
