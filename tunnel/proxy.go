package tunnel

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"
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

// ProxyDestination local connection via IoT secure tunneling.
func ProxyDestination(ctx context.Context, dialer Dialer, endpoint, token string, opts ...ProxyOption) error {
	opt := &ProxyOptions{
		Scheme: "wss",
	}
	for _, o := range opts {
		if err := o(opt); err != nil {
			return err
		}
	}

	wsc, err := websocket.NewConfig(
		fmt.Sprintf("%s://%s/tunnel?local-proxy-mode=destination", opt.Scheme, endpoint),
		fmt.Sprintf("https://%s", endpoint),
	)
	if err != nil {
		return err
	}
	if !opt.NoTLS {
		wsc.TlsConfig = &tls.Config{ServerName: endpoint}
	}
	wsc.Header = http.Header{
		"Access-Token": []string{token},
		"User-Agent":   []string{userAgent},
	}
	wsc.Protocol = append(wsc.Protocol, websocketProtocol)
	ws, err := websocket.DialConfig(wsc)
	if err != nil {
		return err
	}
	ws.PayloadType = websocket.BinaryFrame

	return proxyDestination(ws, dialer, opt.ErrorHandler)
}

// ErrorHandler is an interface to handler error.
type ErrorHandler interface {
	HandleError(error)
}

type errorHandlerFunc func(error)

func (f errorHandlerFunc) HandleError(err error) {
	f(err)
}

// ProxyOption is a type of functional options.
type ProxyOption func(*ProxyOptions) error

// ProxyOptions stores options of the proxy.
type ProxyOptions struct {
	NoTLS        bool
	Scheme       string
	ErrorHandler ErrorHandler
}

// WithErrorHandler sets a ErrorHandler.
func WithErrorHandler(h ErrorHandler) ProxyOption {
	return func(opt *ProxyOptions) error {
		opt.ErrorHandler = h
		return nil
	}
}
