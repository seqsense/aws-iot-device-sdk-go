package tunnel

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/at-wat/mqtt-go"
)

type Tunnel interface {
	mqtt.Handler
	OnError(func(error))
}

type tunnel struct {
	mqtt.ServeMux
	thingName string
	mu        sync.Mutex
	onError   func(err error)
}

func (t *tunnel) topic(operation string) string {
	return "$aws/things/" + t.thingName + "/tunnels/" + operation
}

func New(ctx context.Context, cli mqtt.Client, thingName string) (Tunnel, error) {
	t := &tunnel{
		thingName: thingName,
	}
	if err := t.ServeMux.Handle(t.topic("notify"), mqtt.HandlerFunc(t.notify)); err != nil {
		return nil, err
	}

	err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: t.topic("notify"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *tunnel) notify(msg *mqtt.Message) {
	n := &notification{}
	if err := json.Unmarshal(msg.Payload, n); err != nil {
		t.handleError(err)
		return
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

type ClientMode string

const (
	Source      ClientMode = "source"
	Destination ClientMode = "destination"
)

type notification struct {
	ClientAccessToken string   `json:"clientAccessToken"`
	ClientMode        string   `json:"clientMode"`
	Region            string   `json:"region"`
	Services          []string `json:"services"`
}
