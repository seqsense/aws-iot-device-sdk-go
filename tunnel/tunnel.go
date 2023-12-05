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
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/at-wat/mqtt-go"

	"github.com/seqsense/aws-iot-device-sdk-go/v6"
	"github.com/seqsense/aws-iot-device-sdk-go/v6/internal/ioterr"
)

// Tunnel is an interface of secure tunneling.
type Tunnel interface {
	mqtt.Handler
	OnError(func(error))
}

type tunnel struct {
	mqtt.ServeMux
	thingName string
	mu        sync.Mutex
	onError   func(err error)
	dialerMap map[string]Dialer
	opts      *Options
}

// Options stores options of the tunnel.
type Options struct {
	// EndpointHostFunc is a function returns secure proxy endpoint.
	EndpointHostFunc func(region string) string
	// TopicFunc is a function returns MQTT topic for the operation.
	TopicFunc func(operation string) string

	// ProxyOptions stores slice of ProxyOptions for each service.
	ProxyOptions map[string][]ProxyOption
}

// Option is a type of functional options.
type Option func(*Options) error

// ErrInvalidClientMode indicate that the requested client mode is not valid for the tunnel.
var ErrInvalidClientMode = errors.New("invalid client mode")

func (t *tunnel) topic(operation string) string {
	return "$aws/things/" + t.thingName + "/tunnels/" + operation
}

// New creates new secure tunneling proxy.
func New(ctx context.Context, cli awsiotdev.Device, dialer map[string]Dialer, opts ...Option) (Tunnel, error) {
	t := &tunnel{
		thingName: cli.ThingName(),
		dialerMap: dialer,
	}
	t.opts = &Options{
		TopicFunc:        t.topic,
		EndpointHostFunc: endpointHost,
		ProxyOptions:     make(map[string][]ProxyOption),
	}
	for _, o := range opts {
		if err := o(t.opts); err != nil {
			return nil, ioterr.New(err, "applying options")
		}
	}

	if err := t.ServeMux.Handle(t.opts.TopicFunc("notify"), mqtt.HandlerFunc(t.notify)); err != nil {
		return nil, ioterr.New(err, "registering message handler")
	}

	_, err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: t.opts.TopicFunc("notify"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, ioterr.New(err, "subscribing tunnel topic")
	}
	return t, nil
}

func (t *tunnel) notify(msg *mqtt.Message) {
	n := &Notification{}
	if err := json.Unmarshal(msg.Payload, n); err != nil {
		t.handleError(ioterr.New(err, "unmarshaling notification"))
		return
	}
	if n.ClientMode != Destination {
		t.handleError(ioterr.Newf(ErrInvalidClientMode, "requested %s", n.ClientMode))
		return
	}
	for _, srv := range n.Services {
		srv := srv
		if d, ok := t.dialerMap[srv]; ok {
			go func() {
				opts := append(
					[]ProxyOption{WithErrorHandler(ErrorHandlerFunc(t.handleError))},
					t.opts.ProxyOptions[srv]...,
				)
				err := ProxyDestination(
					d,
					t.opts.EndpointHostFunc(n.Region),
					n.ClientAccessToken,
					opts...,
				)
				if err != nil {
					t.handleError(ioterr.New(err, "creating proxy destination"))
				}
			}()
		}
	}
}

func (t *tunnel) OnError(cb func(err error)) {
	t.mu.Lock()
	t.onError = cb
	t.mu.Unlock()
}

func (t *tunnel) handleError(err error) {
	t.mu.Lock()
	cb := t.onError
	t.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}
