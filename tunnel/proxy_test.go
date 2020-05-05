package tunnel

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

func TestProxyDestination(t *testing.T) {
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
		t.Errorf("payload differs, expected: %s, got: %s", payload1, string(bRecv[:n]))
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
		t.Errorf("message differes, expected: %v, got: %v", msgExpected, m)
	}

	// Check EOF
	if err := tcb.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProxySource(t *testing.T) {
	tca, tcb := net.Pipe()
	ca, cb := net.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		var i int
		err := proxySource(tca,
			acceptFunc(func() (net.Conn, error) {
				if i > 0 {
					return nil, errors.New("done")
				}
				i++
				return cb, nil
			}),
			nil,
		)
		if err != nil {
			t.Error(err)
		}
	}()

	payload1 := "the payload 1"
	payload2 := "the payload 2"

	go func() {
		if _, err := ca.Write([]byte(payload1)); err != nil {
			t.Fatal(err)
		}
	}()

	// Check source to destination
	msgsExpected := []*msg.Message{
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
	for _, me := range msgsExpected {
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
		if !proto.Equal(me, m) {
			t.Errorf("message differs, expected: %v, got: %v", me, m)
		}
	}

	// Check destination to source
	msg := &msg.Message{
		Type:     msg.Message_DATA,
		StreamId: 1,
		Payload:  []byte(payload2),
	}
	b, err := proto.Marshal(msg)
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

	bRecv := make([]byte, 100)
	n, err := ca.Read(bRecv)
	if err != nil {
		t.Fatal(err)
	}
	if string(bRecv[:n]) != payload2 {
		t.Errorf("payload differs, expected: %s, got: %s", payload2, string(bRecv[:n]))
	}

	// Check EOF
	if err := tcb.Close(); err != nil {
		t.Fatal(err)
	}
}

type acceptFunc func() (net.Conn, error)

func (a acceptFunc) Accept() (net.Conn, error) {
	return a()
}

func (acceptFunc) Close() error {
	return nil
}

func (acceptFunc) Addr() net.Addr {
	panic("not implemented")
}
