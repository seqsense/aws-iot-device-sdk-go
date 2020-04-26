package tunnel

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/at-wat/mqtt-go"
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
}

func (t *tunnel) topic(operation string) string {
	return "$aws/things/" + t.thingName + "/tunnels/" + operation
}

// New creates new secure tunneling proxy.
func New(ctx context.Context, cli mqtt.Client, thingName string, dialer map[string]Dialer) (Tunnel, error) {
	t := &tunnel{
		thingName: thingName,
		dialerMap: dialer,
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
