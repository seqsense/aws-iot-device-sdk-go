package tunnel

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func proxyDestination(ws io.ReadWriter, dialer Dialer, eh ErrorHandler) error {
	conns := make(map[int32]io.ReadWriteCloser)
	sz := make([]byte, 2)
	b := make([]byte, 8192)
	for {
		if _, err := io.ReadFull(ws, sz); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
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
			return err
		}
		m := &msg.Message{}
		if err := proto.Unmarshal(b, m); err != nil {
			if eh != nil {
				eh.HandleError(fmt.Errorf("unmarshal failed: %v", err))
			}
			continue
		}
		switch m.Type {
		case msg.Message_STREAM_START:
			conn, err := dialer()
			if err != nil {
				if eh != nil {
					eh.HandleError(fmt.Errorf("dial failed: %v", err))
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
					eh.HandleError(fmt.Errorf("write failed: %v", err))
				}
			}
		}
	}
}
