package tunnel

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

const (
	defaultEndpointHostFormat = "data.tunneling.iot.%s.amazonaws.com"
	defaultPingPeriod         = 5 * time.Second
	websocketProtocol         = "aws.iot.securetunneling-1.0"
	userAgent                 = "aws-iot-device-sdk-go/tunnel"
)

func endpointHost(region string) string {
	return fmt.Sprintf(defaultEndpointHostFormat, region)
}

// Dialer is a proxy destination dialer.
type Dialer func() (io.ReadWriteCloser, error)

// ProxyDestination proxies TCP connection from remote source device to
// the local destination application via IoT secure tunneling.
// This is usually used on IoT things.
func ProxyDestination(dialer Dialer, endpoint, token string, opts ...ProxyOption) error {
	ws, opt, err := openProxyConn(endpoint, "destination", token, opts...)
	if err != nil {
		return err
	}

	pingCancel := newPinger(ws, opt.PingPeriod)
	defer pingCancel()

	return proxyDestination(ws, dialer, opt.ErrorHandler)
}

// ProxySource proxies TCP connection from local socket to
// remote destination application via IoT secure tunneling.
// This is usually used on a computer or bastion server.
func ProxySource(listener net.Listener, endpoint, token string, opts ...ProxyOption) error {
	ws, opt, err := openProxyConn(endpoint, "source", token, opts...)
	if err != nil {
		return err
	}

	pingCancel := newPinger(ws, opt.PingPeriod)
	defer pingCancel()

	return proxySource(ws, listener, opt.ErrorHandler)
}

func openProxyConn(endpoint, mode, token string, opts ...ProxyOption) (*websocket.Conn, *ProxyOptions, error) {
	opt := &ProxyOptions{
		Scheme:     "wss",
		PingPeriod: defaultPingPeriod,
	}
	for _, o := range opts {
		if err := o(opt); err != nil {
			return nil, nil, err
		}
	}

	wsc, err := websocket.NewConfig(
		fmt.Sprintf("%s://%s/tunnel?local-proxy-mode=%s", opt.Scheme, endpoint, mode),
		fmt.Sprintf("https://%s", endpoint),
	)
	if err != nil {
		return nil, nil, err
	}
	if opt.Scheme == "wss" {
		wsc.TlsConfig = &tls.Config{
			ServerName:         endpoint,
			InsecureSkipVerify: opt.InsecureSkipVerify,
		}
	}
	wsc.Header = http.Header{
		"Access-Token": []string{token},
		"User-Agent":   []string{userAgent},
	}
	wsc.Protocol = append(wsc.Protocol, websocketProtocol)
	ws, err := websocket.DialConfig(wsc)
	if err != nil {
		return nil, nil, err
	}
	ws.PayloadType = websocket.BinaryFrame

	return ws, opt, nil
}

// ErrorHandler is an interface to handler error.
type ErrorHandler interface {
	HandleError(error)
}

// ErrorHandlerFunc type is an adapter to use handler function as ErrorHandler.
type ErrorHandlerFunc func(error)

// HandleError implements ErrorHandler interface.
func (f ErrorHandlerFunc) HandleError(err error) {
	f(err)
}

// ProxyOption is a type of functional options.
type ProxyOption func(*ProxyOptions) error

// ProxyOptions stores options of the proxy.
type ProxyOptions struct {
	InsecureSkipVerify bool
	Scheme             string
	ErrorHandler       ErrorHandler
	PingPeriod         time.Duration
}

// WithErrorHandler sets a ErrorHandler.
func WithErrorHandler(h ErrorHandler) ProxyOption {
	return func(opt *ProxyOptions) error {
		opt.ErrorHandler = h
		return nil
	}
}

// WithPingPeriod sets ping send period.
func WithPingPeriod(d time.Duration) ProxyOption {
	return func(opt *ProxyOptions) error {
		opt.PingPeriod = d
		return nil
	}
}
