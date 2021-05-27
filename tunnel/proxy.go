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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

const (
	defaultEndpointHostFormat = "data.tunneling.iot.%s.amazonaws.com"
	defaultPingPeriod         = 5 * time.Second
	websocketProtocol         = "aws.iot.securetunneling-1.0"
	userAgent                 = "aws-iot-device-sdk-go/tunnel"
)

// ErrUnsupportedSchema indicate that the requested protocol schema is not supported.
var ErrUnsupportedSchema = errors.New("unsupported schema")

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
		return ioterr.New(err, "opening proxy destination")
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
		return ioterr.New(err, "opening proxy source")
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
			return nil, nil, ioterr.New(err, "applying options")
		}
	}

	var defaultPort int
	switch opt.Scheme {
	case "wss":
		defaultPort = 443
	case "ws":
		defaultPort = 80
	default:
		return nil, nil, ioterr.New(ErrUnsupportedSchema, opt.Scheme)
	}

	wsc, err := websocket.NewConfig(
		fmt.Sprintf("%s://%s/tunnel?local-proxy-mode=%s", opt.Scheme, endpoint, mode),
		fmt.Sprintf("https://%s", endpoint),
	)
	if err != nil {
		return nil, nil, ioterr.New(err, "creating ws config")
	}
	if opt.Scheme == "wss" {
		wsc.TlsConfig = &tls.Config{
			// Remove protocol default port number from the URI to avoid TLS certificate validation error.
			ServerName:         strings.TrimSuffix(endpoint, fmt.Sprintf(":%d", defaultPort)),
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
		return nil, nil, ioterr.New(err, "creating ws dial config")
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
