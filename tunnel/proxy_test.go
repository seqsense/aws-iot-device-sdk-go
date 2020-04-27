package tunnel

import (
	"io"
	"net"
	"sync"
	"testing"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func TestProxyImpl(t *testing.T) {
	chWsSend := make(chan []byte)
	chWsReceive := make(chan []byte)
	ca, cb := net.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		tn := &tunnel{}
		err := tn.proxyImpl(nil,
			&mockCodec{
				send: func(ws *websocket.Conn, v interface{}) error {
					chWsSend <- v.([]byte)
					return nil
				},
				receive: func(ws *websocket.Conn, v interface{}) error {
					var ok bool
					*(v.(*[]byte)), ok = <-chWsReceive
					if !ok {
						return io.EOF
					}
					return nil
				},
			},
			func() (io.ReadWriteCloser, error) {
				return cb, nil
			},
		)
		if err != nil {
			t.Error(err)
		}
	}()

	payload1 := "the payload 1"
	payload2 := "the payload 2"

	// Check source to destination
	msgs := []*msg.Message{
		{
			Type:     msg.Message_STREAM_START,
			StreamId: 1,
		},
		{
			Type:     msg.Message_DATA,
			StreamId: 1,
			Payload:  []byte(payload1),
		},
	}
	for _, m := range msgs {
		b, err := proto.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		l := len(b)
		chWsReceive <- append(
			[]byte{byte(l >> 8), byte(l)},
			b...,
		)
	}
	bRecv := make([]byte, 100)
	n, err := ca.Read(bRecv)
	if err != nil {
		t.Fatal(err)
	}
	if string(bRecv[:n]) != payload1 {
		t.Errorf("Payload differs, expected: %s, got: %s", payload1, string(bRecv[:n]))
	}

	// Check destination to source
	if _, err := ca.Write([]byte(payload2)); err != nil {
		t.Fatal(err)
	}
	bSent := <-chWsSend
	m := &msg.Message{}
	if err := proto.Unmarshal(bSent[2:], m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	msgExpected := &msg.Message{
		Type:     msg.Message_DATA,
		StreamId: 1,
		Payload:  []byte(payload2),
	}
	if !proto.Equal(msgExpected, m) {
		t.Errorf("Payload differes, expected: %v, got: %v", msgExpected, m)
	}

	// Check EOF
	close(chWsReceive)
}

type mockCodec struct {
	send    func(ws *websocket.Conn, v interface{}) error
	receive func(ws *websocket.Conn, v interface{}) error
}

func (c *mockCodec) Send(ws *websocket.Conn, v interface{}) error {
	return c.send(ws, v)
}

func (c *mockCodec) Receive(ws *websocket.Conn, v interface{}) error {
	return c.receive(ws, v)
}
