package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/at-wat/mqtt-go"
)

// Shadow is an interface of Thing Shadow.
type Shadow interface {
	mqtt.Handler
	// Get thing state.
	Get(ctx context.Context) error
	// Report thing state.
	Report(ctx context.Context, state interface{}) error
	// Desire sets desired thing state.
	Desire(ctx context.Context, state interface{}) error
	// Document returns full thing document.
	Document() *ThingDocument
	// Delete thing shadow.
	Delete(ctx context.Context) error
	// OnDelta sets handler of state deltas.
	OnDelta(func(delta map[string]interface{}))
	// OnError sets handler of asynchronous errors.
	OnError(func(error))
}

type shadow struct {
	mqtt.ServeMux
	cli       mqtt.Client
	thingName string
	doc       *ThingDocument
	mu        sync.Mutex
	onDelta   func(delta map[string]interface{})
	onError   func(err error)
}

func (s *shadow) topic(operation string) string {
	return "$aws/things/" + s.thingName + "/shadow/" + operation
}

func (s *shadow) getAccepted(msg *mqtt.Message) {
	doc := &ThingDocument{}
	if err := json.Unmarshal(msg.Payload, doc); err != nil {
		s.handleError(err)
		return
	}
	s.doc = doc
	s.handleDelta(s.doc.State.Delta)
}

func (s *shadow) getRejected(msg *mqtt.Message) {
	s.handleError(fmt.Errorf("%s: %s", msg.Topic, string(msg.Payload)))
}

func (s *shadow) updateAccepted(msg *mqtt.Message) {
	doc := &thingDocumentRaw{}
	if err := json.Unmarshal(msg.Payload, doc); err != nil {
		s.handleError(err)
		return
	}
	if err := s.doc.update(doc); err != nil {
		s.handleError(err)
		return
	}
}

func (s *shadow) updateDelta(msg *mqtt.Message) {
	state := &thingDelta{}
	if err := json.Unmarshal(msg.Payload, state); err != nil {
		s.handleError(err)
		return
	}
	ok, err := s.doc.updateDelta(state)
	if err != nil {
		s.handleError(err)
		return
	}
	if ok {
		s.handleDelta(s.doc.State.Delta)
	}
}

func (s *shadow) updateRejected(msg *mqtt.Message) {
	s.handleError(fmt.Errorf("%s: %s", msg.Topic, string(msg.Payload)))
}

func (s *shadow) deleteAccepted(msg *mqtt.Message) {
	s.doc = nil
	s.handleDelta(s.doc.State.Delta)
}

func (s *shadow) deleteRejected(msg *mqtt.Message) {
	s.handleError(fmt.Errorf("%s: %s", msg.Topic, string(msg.Payload)))
}

// New creates Thing Shadow interface.
func New(ctx context.Context, cli mqtt.Client, thingName string) (Shadow, error) {
	s := &shadow{
		cli:       cli,
		thingName: thingName,
	}
	for _, sub := range []struct {
		topic   string
		handler mqtt.Handler
	}{
		{s.topic("update/delta"), mqtt.HandlerFunc(s.updateDelta)},
		{s.topic("update/accepted"), mqtt.HandlerFunc(s.updateAccepted)},
		{s.topic("update/rejected"), mqtt.HandlerFunc(s.updateRejected)},
		{s.topic("get/accepted"), mqtt.HandlerFunc(s.getAccepted)},
		{s.topic("get/rejected"), mqtt.HandlerFunc(s.getRejected)},
		{s.topic("delete/accepted"), mqtt.HandlerFunc(s.deleteAccepted)},
		{s.topic("delete/rejected"), mqtt.HandlerFunc(s.deleteRejected)},
	} {
		if err := s.ServeMux.Handle(sub.topic, sub.handler); err != nil {
			return nil, err
		}
	}

	err := cli.Subscribe(ctx,
		mqtt.Subscription{Topic: s.topic("update/delta"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("update/accepted"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("update/rejected"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("get/accepted"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("get/rejected"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("delete/accepted"), QoS: mqtt.QoS1},
		mqtt.Subscription{Topic: s.topic("delete/rejected"), QoS: mqtt.QoS1},
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *shadow) Report(ctx context.Context, state interface{}) error {
	rawState, err := json.Marshal(state)
	if err != nil {
		return err
	}
	rawStateJSON := json.RawMessage(rawState)
	data, err := json.Marshal(&thingDocumentRaw{
		State: thingStateRaw{Reported: rawStateJSON},
	})
	if err != nil {
		return err
	}
	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("update"),
		QoS:     mqtt.QoS1,
		Payload: data,
	}); err != nil {
		return err
	}
	return nil
}

func (s *shadow) Desire(ctx context.Context, state interface{}) error {
	rawState, err := json.Marshal(state)
	if err != nil {
		return err
	}
	rawStateJSON := json.RawMessage(rawState)
	data, err := json.Marshal(&thingDocumentRaw{
		State: thingStateRaw{Desired: rawStateJSON},
	})
	if err != nil {
		return err
	}
	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("update"),
		QoS:     mqtt.QoS1,
		Payload: data,
	}); err != nil {
		return err
	}
	return nil
}

func (s *shadow) Get(ctx context.Context) error {
	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("get"),
		QoS:     mqtt.QoS1,
		Payload: []byte{},
	}); err != nil {
		return err
	}
	return nil
}

func (s *shadow) Delete(ctx context.Context) error {
	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("delete"),
		QoS:     mqtt.QoS1,
		Payload: []byte{},
	}); err != nil {
		return err
	}
	return nil
}

func (s *shadow) Document() *ThingDocument {
	return s.doc
}

func (s *shadow) OnDelta(cb func(delta map[string]interface{})) {
	s.mu.Lock()
	s.onDelta = cb
	s.mu.Unlock()
}

func (s *shadow) handleDelta(delta map[string]interface{}) {
	s.mu.Lock()
	cb := s.onDelta
	s.mu.Unlock()
	if cb != nil {
		cb(delta)
	}
}

func (s *shadow) OnError(cb func(err error)) {
	s.mu.Lock()
	s.onError = cb
	s.mu.Unlock()
}

func (s *shadow) handleError(err error) {
	s.mu.Lock()
	cb := s.onError
	s.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}
