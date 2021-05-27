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
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
	"github.com/seqsense/aws-iot-device-sdk-go/v5/tunnel/msg"
)

var (
	errConnect = errors.New("conn error")
)

func TestProxyDestination(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tca, tcb := net.Pipe()
		ca, cb := net.Pipe()

		var wg sync.WaitGroup
		wg.Add(1)
		defer wg.Wait()

		go func() {
			defer wg.Done()
			err := proxyDestination(tca,
				func() (io.ReadWriteCloser, error) { return cb, nil },
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

		if err := tcb.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("DialError", func(t *testing.T) {
		tca, tcb := net.Pipe()

		var wg sync.WaitGroup
		defer wg.Wait()

		chErr := make(chan error)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxyDestination(tca,
				func() (io.ReadWriteCloser, error) { return nil, errConnect },
				ErrorHandlerFunc(func(err error) {
					chErr <- err
				}),
			); err != nil {
				t.Error(err)
			}
		}()

		b, err := proto.Marshal(&msg.Message{
			Type:     msg.Message_STREAM_START,
			StreamId: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		l := len(b)
		if _, err = tcb.Write(append([]byte{byte(l >> 8), byte(l)}, b...)); err != nil {
			t.Fatal(err)
		}
		tcb.Close()

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, errConnect) {
				t.Errorf("Expected error: %v, got: %v", errConnect, err)
			}
		}
	})
	t.Run("WriteError", func(t *testing.T) {
		tca, tcb := net.Pipe()
		ca, cb := net.Pipe()

		var wg sync.WaitGroup
		defer wg.Wait()

		chErr := make(chan error)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxyDestination(tca,
				func() (io.ReadWriteCloser, error) { return cb, nil },
				ErrorHandlerFunc(func(err error) {
					chErr <- err
				}),
			); err != nil {
				t.Error(err)
			}
		}()

		b, err := proto.Marshal(&msg.Message{
			Type:     msg.Message_STREAM_START,
			StreamId: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		l := len(b)
		if _, err = tcb.Write(append([]byte{byte(l >> 8), byte(l)}, b...)); err != nil {
			t.Fatal(err)
		}
		ca.Close()

		b, err = proto.Marshal(&msg.Message{
			Type:     msg.Message_DATA,
			StreamId: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		l = len(b)
		if _, err = tcb.Write(append([]byte{byte(l >> 8), byte(l)}, b...)); err != nil {
			t.Fatal(err)
		}
		tcb.Close()

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, io.ErrClosedPipe) {
				t.Errorf("Expected error: %v, got: %v", io.ErrClosedPipe, err)
			}
		}
	})
	t.Run("UnexpectedEOF", func(t *testing.T) {
		packets := map[string][]byte{
			"LengthHeader": []byte{0x00},
			"Contents":     []byte{0x00, 0x10, 0x00},
		}
		for name, packet := range packets {
			packet := packet
			t.Run(name, func(t *testing.T) {
				ca, cb := net.Pipe()

				go func() {
					if _, err := cb.Write(packet); err != nil {
						t.Error(err)
					}
					if err := cb.Close(); err != nil {
						t.Error(err)
					}
				}()

				err := proxyDestination(ca, nil, nil)

				var ie *ioterr.Error
				if !errors.As(err, &ie) {
					t.Errorf("Expected error type: %T, got: %T", ie, err)
				}
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("Expected error: %v, got: %v", io.ErrUnexpectedEOF, err)
				}
			})
		}
	})
	t.Run("InvalidProto", func(t *testing.T) {
		ca, cb := net.Pipe()

		packet := []byte{0, 3, 0xff, 0xff, 0xff}

		chErr := make(chan error)
		go func() {
			if err := proxyDestination(ca, nil,
				ErrorHandlerFunc(func(err error) { chErr <- err }),
			); err != nil {
				t.Error(err)
			}
		}()

		if _, err := cb.Write(packet); err != nil {
			t.Error(err)
		}
		if err := cb.Close(); err != nil {
			t.Error(err)
		}

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, proto.Error) {
				t.Errorf("Expected error: %v, got: %v", proto.Error, err)
			}
		}
	})
}

func TestProxySource(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tca, tcb := net.Pipe()
		ca, cb := net.Pipe()

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var i int
			if err := proxySource(tca,
				acceptFunc(func() (net.Conn, error) {
					if i > 0 {
						return nil, errors.New("done")
					}
					i++
					return cb, nil
				}),
				nil,
			); err != nil {
				t.Error(err)
			}
		}()

		payload1 := "the payload 1"
		payload2 := "the payload 2"

		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ca.Write([]byte(payload1)); err != nil {
				t.Error(err)
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
	})
	t.Run("AcceptError", func(t *testing.T) {
		tca, tcb := net.Pipe()

		var wg sync.WaitGroup
		defer wg.Wait()

		chErr := make(chan error)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proxySource(tca,
				acceptFunc(func() (net.Conn, error) { return nil, errConnect }),
				ErrorHandlerFunc(func(err error) {
					chErr <- err
				}),
			); err != nil {
				t.Error(err)
			}
		}()

		// Check destination to source
		msg := &msg.Message{
			Type:     msg.Message_DATA,
			StreamId: 1,
			Payload:  []byte("test"),
		}
		b, err := proto.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		l := len(b)
		if _, err = tcb.Write(append([]byte{byte(l >> 8), byte(l)}, b...)); err != nil {
			t.Fatal(err)
		}

		if err := tcb.Close(); err != nil {
			t.Fatal(err)
		}

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, errConnect) {
				t.Errorf("Expected error: %v, got: %v", errConnect, err)
			}
		}
	})
	t.Run("UnexpectedEOF", func(t *testing.T) {
		packets := map[string][]byte{
			"LengthHeader": []byte{0x00},
			"Contents":     []byte{0x00, 0x10, 0x00},
		}
		for name, packet := range packets {
			packet := packet
			t.Run(name, func(t *testing.T) {
				ca, cb := net.Pipe()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					if _, err := cb.Write(packet); err != nil {
						t.Error(err)
					}
					if err := cb.Close(); err != nil {
						t.Error(err)
					}
				}()

				err := proxySource(ca,
					acceptFunc(func() (net.Conn, error) {
						wg.Wait()
						return nil, errConnect
					}),
					nil,
				)

				var ie *ioterr.Error
				if !errors.As(err, &ie) {
					t.Errorf("Expected error type: %T, got: %T", ie, err)
				}
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("Expected error: %v, got: %v", io.ErrUnexpectedEOF, err)
				}
			})
		}
	})
	t.Run("InvalidProto", func(t *testing.T) {
		ca, cb := net.Pipe()

		packet := []byte{0, 3, 0xff, 0xff, 0xff}

		var wg sync.WaitGroup
		wg.Add(1)

		chErr := make(chan error)
		go func() {
			if err := proxySource(ca,
				acceptFunc(func() (net.Conn, error) {
					wg.Wait()
					return nil, errConnect
				}),
				ErrorHandlerFunc(func(err error) { chErr <- err }),
			); err != nil {
				t.Error(err)
			}
		}()

		if _, err := cb.Write(packet); err != nil {
			t.Error(err)
		}
		if err := cb.Close(); err != nil {
			t.Error(err)
		}

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, proto.Error) {
				t.Errorf("Expected error: %v, got: %v", proto.Error, err)
			}
		}

		wg.Done()
	})
	t.Run("WriteError", func(t *testing.T) {
		tca, tcb := net.Pipe()
		_, cb := net.Pipe()
		defer cb.Close()

		var wg sync.WaitGroup
		defer wg.Wait()

		chAccept := make(chan struct{})
		chDone := make(chan struct{})
		chErr := make(chan error)

		wg.Add(1)
		go func() {
			defer wg.Done()
			var i int
			if err := proxySource(tca,
				acceptFunc(func() (net.Conn, error) {
					if i > 0 {
						<-chDone
						return nil, errors.New("done")
					}
					i++
					defer func() {
						close(chAccept)
					}()
					return cb, nil
				}),
				ErrorHandlerFunc(func(err error) { chErr <- err }),
			); err != nil {
				t.Error(err)
			}
		}()

		<-chAccept
		if err := tcb.Close(); err != nil {
			t.Fatal(err)
		}

		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		case err := <-chErr:
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
			}
			if !errors.Is(err, io.ErrClosedPipe) {
				t.Errorf("Expected error: %v, got: %v", io.ErrClosedPipe, err)
			}
		}
		close(chDone)
		<-chErr // Error on accept error
	})
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

func TestProxyOption_validate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		opts := []*ProxyOptions{
			{Scheme: "ws"},
			{Scheme: "wss"},
		}
		for _, o := range opts {
			o := o
			t.Run(o.Scheme, func(t *testing.T) {
				if err := o.validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			})
		}
	})
	t.Run("Invalid", func(t *testing.T) {
		opts := []*ProxyOptions{
			{Scheme: "http"},
			{Scheme: ""},
		}
		for _, o := range opts {
			o := o
			t.Run(o.Scheme, func(t *testing.T) {
				if err := o.validate(); err == nil {
					t.Error("Validation must fail")
				}
			})
		}
	})
}
