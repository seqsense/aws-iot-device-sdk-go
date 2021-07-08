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
	"io"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
	"github.com/seqsense/aws-iot-device-sdk-go/v6/tunnel/msg"
)

func proxySource(ws io.ReadWriter, listener net.Listener, eh ErrorHandler, stat Stat) error {
	var muConns sync.Mutex
	conns := make(map[int32]io.ReadWriteCloser)
	go func() {
		var streamID int32 = 1
		for {
			conn, err := listener.Accept()
			if err != nil {
				if eh != nil {
					eh.HandleError(ioterr.New(err, "accepting source connection"))
				}
				return
			}

			muConns.Lock()
			id := streamID
			conns[id] = conn
			streamID++
			muConns.Unlock()

			if err := msg.WriteMessage(ws, &msg.Message{
				Type:     msg.Message_STREAM_START,
				StreamId: id,
			}); err != nil {
				if eh != nil {
					eh.HandleError(ioterr.New(err, "sending message"))
				}
				continue
			}

			go readProxy(ws, conn, id, eh)
		}
	}()

	sz := make([]byte, 2)
	b := make([]byte, 8192)
	for {
		if _, err := io.ReadFull(ws, sz); err != nil {
			if err == io.EOF {
				return nil
			}
			return ioterr.New(err, "reading length header")
		}
		l := int(sz[0])<<8 | int(sz[1])
		if cap(b) < l {
			b = make([]byte, l)
		}
		b = b[:l]
		if _, err := io.ReadFull(ws, b); err != nil {
			if err == io.EOF {
				return nil
			}
			return ioterr.New(err, "reading message")
		}
		m := &msg.Message{}
		if err := proto.Unmarshal(b, m); err != nil {
			if eh != nil {
				eh.HandleError(ioterr.New(err, "unmarshaling message"))
			}
			continue
		}
		switch m.Type {
		case msg.Message_STREAM_RESET:
			muConns.Lock()
			if conn, ok := conns[m.StreamId]; ok {
				_ = conn.Close()
				delete(conns, m.StreamId)
			}
			muConns.Unlock()

		case msg.Message_SESSION_RESET:
			muConns.Lock()
			for id, c := range conns {
				_ = c.Close()
				delete(conns, id)
			}
			muConns.Unlock()
			return io.EOF

		case msg.Message_DATA:
			muConns.Lock()
			conn, ok := conns[m.StreamId]
			muConns.Unlock()
			if ok {
				if _, err := conn.Write(m.Payload); err != nil {
					eh.HandleError(ioterr.New(err, "writing message"))
				}
			}
		}

		if stat != nil {
			stat.Update(func(stat *Statistics) {
				stat.NumConn = len(conns)
			})
		}
	}
}
