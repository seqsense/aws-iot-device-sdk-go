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

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
	"github.com/seqsense/aws-iot-device-sdk-go/v6/tunnel/msg"
)

func proxyDestination(ws io.ReadWriter, dialer Dialer, eh ErrorHandler, stat Stat) error {
	conns := make(map[int32]io.ReadWriteCloser)
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
		case msg.Message_STREAM_START:
			conn, err := dialer()
			if err != nil {
				if eh != nil {
					eh.HandleError(ioterr.New(err, "dialing to destination"))
				}
				continue
			}

			conns[m.StreamId] = conn
			go readProxy(ws, conn, m.StreamId, eh)

		case msg.Message_STREAM_RESET:
			if conn, ok := conns[m.StreamId]; ok {
				_ = conn.Close()
				delete(conns, m.StreamId)
			}

		case msg.Message_SESSION_RESET:
			for id, c := range conns {
				_ = c.Close()
				delete(conns, id)
			}
			return io.EOF

		case msg.Message_DATA:
			if conn, ok := conns[m.StreamId]; ok {
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
