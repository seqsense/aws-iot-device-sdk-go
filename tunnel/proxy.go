package tunnel

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/msg"
)

const (
	defaultEndpointHostFormat = "data.tunneling.iot.%s.amazonaws.com"
	websocketProtocol         = "aws.iot.securetunneling-1.0"
	userAgent                 = "aws-iot-device-sdk-go/tunnel"
)

func endpointHost(region string) string {
	return fmt.Sprintf(defaultEndpointHostFormat, region)
}

// Dialer is a proxy destination dialer.
type Dialer func() (io.ReadWriteCloser, error)

type websocketCodec interface {
	Send(*websocket.Conn, interface{}) error
	Receive(*websocket.Conn, interface{}) error
}

func proxy(ctx context.Context, dialer Dialer, notification *notification, opts ...proxyOption) error {
	if notification.ClientMode != Destination {
		return errors.New("unsupported client mode")
	}

	opt := &proxyOpt{
		scheme: "wss",
	}
	for _, o := range opts {
		if err := o(opt); err != nil {
			return err
		}
	}

	host := endpointHost(notification.Region)
	wsc, err := websocket.NewConfig(
		fmt.Sprintf("%s://%s/tunnel?local-proxy-mode=destination", opt.scheme, host),
		fmt.Sprintf("https://%s", host),
	)
	if err != nil {
		return err
	}
	if !opt.noTLS {
		wsc.TlsConfig = &tls.Config{ServerName: host}
	}
	wsc.Header = http.Header{
		"access-token": []string{notification.ClientAccessToken},
		"User-Agent":   []string{userAgent},
	}
	wsc.Protocol = append(wsc.Protocol, websocketProtocol)
	ws, err := websocket.DialConfig(wsc)
	if err != nil {
		return err
	}
	ws.PayloadType = websocket.BinaryFrame

	return proxyImpl(ws, websocket.Message, dialer)
}

func proxyImpl(ws *websocket.Conn, codec websocketCodec, dialer Dialer) error {
	conns := make(map[int32]io.ReadWriteCloser)
	for {
		var b []byte
		err := codec.Receive(ws, &b)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(b) < 2 {
			log.Printf("discarded short packet")
			continue
		}
		m := &msg.Message{}
		if err := proto.Unmarshal(b[2:], m); err != nil {
			log.Printf("unmarshal failed: %v", err)
			continue
		}
		switch m.Type {
		case msg.Message_STREAM_START:
			conn, err := dialer()
			if err != nil {
				log.Printf("dial failed: %v", err)
				continue
			}

			conns[m.StreamId] = conn
			go func() {
				b := make([]byte, 8192)
				for {
					n, err := conn.Read(b)
					if err != nil {
						if err == io.EOF {
							return
						}
						log.Printf("connection closed: %v", err)
						return
					}
					ms := &msg.Message{
						Type:     msg.Message_DATA,
						StreamId: m.StreamId,
						Payload:  b[:n],
					}
					bs, err := proto.Marshal(ms)
					if err != nil {
						log.Printf("marshal failed: %v", err)
						continue
					}
					l := len(bs)
					if err := codec.Send(ws,
						append([]byte{
							byte(l >> 8), byte(l),
						}, bs...),
					); err != nil {
						log.Printf("send failed: %v", err)
						return
					}
				}
			}()

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
					log.Printf("write failed: %v", err)
				}
			}
		}
	}
}

type proxyOption func(*proxyOpt) error

type proxyOpt struct {
	noTLS  bool
	scheme string
}
