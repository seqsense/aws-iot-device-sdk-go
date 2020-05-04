package tunnel

import (
	"context"
	"crypto/tls"
	"errors"
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

// Proxy local connection via IoT secure tunneling.
func Proxy(ctx context.Context, dialer Dialer, notification *Notification, opts ...proxyOption) error {
	if notification.ClientMode != Destination {
		return errors.New("unsupported client mode")
	}

	opt := &ProxyOptions{
		Scheme:           "wss",
		EndpointHostFunc: endpointHost,
	}
	for _, o := range opts {
		if err := o(opt); err != nil {
			return err
		}
	}

	host := opt.EndpointHostFunc(notification.Region)
	wsc, err := websocket.NewConfig(
		fmt.Sprintf("%s://%s/tunnel?local-proxy-mode=destination", opt.Scheme, host),
		fmt.Sprintf("https://%s", host),
	)
	if err != nil {
		return err
	}
	if !opt.NoTLS {
		wsc.TlsConfig = &tls.Config{ServerName: host}
	}
	wsc.Header = http.Header{
		"Access-Token": []string{notification.ClientAccessToken},
		"User-Agent":   []string{userAgent},
	}
	wsc.Protocol = append(wsc.Protocol, websocketProtocol)
	ws, err := websocket.DialConfig(wsc)
	if err != nil {
		return err
	}
	ws.PayloadType = websocket.BinaryFrame

	return proxyDestinationImpl(ws, dialer, opt.ErrorHandler)
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
	NoTLS            bool
	Scheme           string
	EndpointHostFunc func(string) string
	ErrorHandler     ErrorHandler
}

// WithEndpointHostFunc sets a function to return proxy endpoint host.
func WithEndpointHostFunc(f func(region string) string) ProxyOption {
	return func(opt *ProxyOptions) error {
		opt.EndpointHostFunc = f
		return nil
	}
}

// WithErrorHandler sets a ErrorHandler.
func WithErrorHandler(h ErrorHandler) ProxyOption {
	return func(opt *ProxyOptions) error {
		opt.ErrorHandler = h
		return nil
	}
}
