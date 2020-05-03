package tunnel

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4"
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
}

// Option is a type of functional options.
type Option func(*Options) error

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
	}
	for _, o := range opts {
		if err := o(t.opts); err != nil {
			return nil, err
		}
	}

	if err := t.ServeMux.Handle(t.opts.TopicFunc("notify"), mqtt.HandlerFunc(t.notify)); err != nil {
		return nil, err
	}

	err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: t.opts.TopicFunc("notify"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *tunnel) notify(msg *mqtt.Message) {
	n := &Notification{}
	if err := json.Unmarshal(msg.Payload, n); err != nil {
		t.handleError(err)
		return
	}
	for _, srv := range n.Services {
		if d, ok := t.dialerMap[srv]; ok {
			go func() {
				err := t.proxy(context.Background(), d, n)
				if err != nil {
					t.handleError(err)
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
