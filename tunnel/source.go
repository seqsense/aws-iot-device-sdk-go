package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func proxySource(ws io.ReadWriter, listener net.Listener, eh ErrorHandler) error {
	var muConns sync.Mutex
	conns := make(map[int32]io.ReadWriteCloser)
	go func() {
		var streamID int32
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Exit main routine
				return
			}

			m := &msg.Message{
				Type:     msg.Message_STREAM_START,
				StreamId: streamID,
			}
			bs, err := proto.Marshal(m)
			if err != nil {
				if eh != nil {
					eh.HandleError(fmt.Errorf("marshal failed: %v", err))
				}
				continue
			}
			l := len(bs)
			if _, err := ws.Write(
				append([]byte{
					byte(l >> 8), byte(l),
				}, bs...),
			); err != nil {
				if eh != nil {
					eh.HandleError(fmt.Errorf("send failed: %v", err))
				}
				continue
			}

			muConns.Lock()
			conns[streamID] = conn
			streamID++
			muConns.Unlock()

			go readProxy(ws, conn, m.StreamId, eh)
		}
	}()

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
					eh.HandleError(fmt.Errorf("write failed: %v", err))
				}
			}
		}
	}
}
