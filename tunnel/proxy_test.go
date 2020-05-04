package tunnel

import (
	"io"
	"net"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func TestProxyImpl(t *testing.T) {
	tca, tcb := net.Pipe()
	ca, cb := net.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		err := proxyDestination(tca,
			func() (io.ReadWriteCloser, error) {
				return cb, nil
			},
			nil,
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
		_, err = tcb.Write(append(
			[]byte{byte(l >> 8), byte(l)},
			b...,
		))
		if err != nil {
			t.Fatal(err)
		}
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
	sz := make([]byte, 2)
	if _, err := io.ReadFull(tcb, sz); err != nil {
		t.Fatal(err)
	}
	bSent := make([]byte, int(sz[0])<<8|int(sz[1]))
	if _, err := io.ReadFull(tcb, bSent); err != nil {
		t.Fatal(err)
	}
	m := &msg.Message{}
	if err := proto.Unmarshal(bSent, m); err != nil {
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
	if err := tcb.Close(); err != nil {
		t.Fatal(err)
	}
}
