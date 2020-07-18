package shadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/at-wat/mqtt-go"
	"github.com/seqsense/aws-iot-device-sdk-go/v4"
)

// Shadow is an interface of Thing Shadow.
type Shadow interface {
	mqtt.Handler
	// Get thing state and update local state document.
	Get(ctx context.Context) (*ThingDocument, error)
	// Report thing state and update local state document.
	Report(ctx context.Context, state interface{}) (*ThingDocument, error)
	// Desire sets desired thing state and update local state document.
	Desire(ctx context.Context, state interface{}) (*ThingDocument, error)
	// Document returns full thing document.
	Document() *ThingDocument
	// Delete thing shadow.
	Delete(ctx context.Context) error
	// OnDelta sets handler of state deltas.
	OnDelta(func(delta map[string]interface{}))
	// OnError sets handler of asynchronous errors.
	OnError(func(error))
}

// ErrRejected is returned if AWS IoT responded on rejected topic.
var ErrRejected = errors.New("rejected")

// ErrInvalidResponse is returned if failed to parse response from AWS IoT.
var ErrInvalidResponse = errors.New("invalid response from AWS IoT")

type shadow struct {
	mqtt.ServeMux
	cli       mqtt.Client
	thingName string
	doc       *ThingDocument
	mu        sync.Mutex
	onDelta   func(delta map[string]interface{})
	onError   func(err error)

	chResps map[string]chan interface{}

	msgToken uint32
}

func (s *shadow) token() string {
	token := atomic.AddUint32(&s.msgToken, 1)
	return fmt.Sprintf("%x", token)
}

func (s *shadow) topic(operation string) string {
	return "$aws/things/" + s.thingName + "/shadow/" + operation
}

func (s *shadow) handleResponse(r interface{}) {
	token, ok := clientToken(r)
	if !ok {
		return
	}
	s.mu.Lock()
	ch, ok := s.chResps[token]
	s.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- r:
	default:
	}
}

func (s *shadow) getAccepted(msg *mqtt.Message) {
	doc := &ThingDocument{}
	if err := json.Unmarshal(msg.Payload, doc); err != nil {
		s.handleError(err)
		return
	}
	s.doc = doc
	s.handleResponse(doc)

	s.handleDelta(s.doc.State.Delta)
}

func (s *shadow) rejected(msg *mqtt.Message) {
	e := &ErrorResponse{}
	if err := json.Unmarshal(msg.Payload, e); err != nil {
		s.handleError(err)
		return
	}
	s.handleResponse(e)
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
	s.handleResponse(doc)
}

func (s *shadow) updateDelta(msg *mqtt.Message) {
	state := &thingDelta{}
	if err := json.Unmarshal(msg.Payload, state); err != nil {
		s.handleError(err)
		return
	}
	ok := s.doc.updateDelta(state)
	if ok {
		s.handleDelta(s.doc.State.Delta)
	}
}

func (s *shadow) deleteAccepted(msg *mqtt.Message) {
	doc := &thingDocumentRaw{}
	if err := json.Unmarshal(msg.Payload, doc); err != nil {
		s.handleError(err)
		return
	}
	s.doc = nil
	s.handleResponse(doc)
}

// New creates Thing Shadow interface.
func New(ctx context.Context, cli awsiotdev.Device) (Shadow, error) {
	s := &shadow{
		cli:       cli,
		thingName: cli.ThingName(),
		doc: &ThingDocument{
			State: ThingState{
				Desired:  map[string]interface{}{},
				Reported: map[string]interface{}{},
				Delta:    map[string]interface{}{},
			},
		},

		chResps: make(map[string]chan interface{}),
	}
	for _, sub := range []struct {
		topic   string
		handler mqtt.Handler
	}{
		{s.topic("update/delta"), mqtt.HandlerFunc(s.updateDelta)},
		{s.topic("update/accepted"), mqtt.HandlerFunc(s.updateAccepted)},
		{s.topic("update/rejected"), mqtt.HandlerFunc(s.rejected)},
		{s.topic("get/accepted"), mqtt.HandlerFunc(s.getAccepted)},
		{s.topic("get/rejected"), mqtt.HandlerFunc(s.rejected)},
		{s.topic("delete/accepted"), mqtt.HandlerFunc(s.deleteAccepted)},
		{s.topic("delete/rejected"), mqtt.HandlerFunc(s.rejected)},
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

func (s *shadow) Report(ctx context.Context, state interface{}) (*ThingDocument, error) {
	rawState, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	token := s.token()
	rawStateJSON := json.RawMessage(rawState)
	data, err := json.Marshal(&thingDocumentRaw{
		State:       thingStateRaw{Reported: rawStateJSON},
		ClientToken: token,
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{}, 1)
	s.mu.Lock()
	s.chResps[token] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.chResps, token)
		s.mu.Unlock()
	}()

	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("update"),
		QoS:     mqtt.QoS1,
		Payload: data,
	}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		switch r := res.(type) {
		case *thingDocumentRaw:
			return s.doc, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ErrInvalidResponse
		}
	}
}

func (s *shadow) Desire(ctx context.Context, state interface{}) (*ThingDocument, error) {
	rawState, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	token := s.token()
	rawStateJSON := json.RawMessage(rawState)
	data, err := json.Marshal(&thingDocumentRaw{
		State:       thingStateRaw{Desired: rawStateJSON},
		ClientToken: token,
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{}, 1)
	s.mu.Lock()
	s.chResps[token] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.chResps, token)
		s.mu.Unlock()
	}()

	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("update"),
		QoS:     mqtt.QoS1,
		Payload: data,
	}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		switch r := res.(type) {
		case *thingDocumentRaw:
			return s.doc, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ErrInvalidResponse
		}
	}
}

func (s *shadow) Get(ctx context.Context) (*ThingDocument, error) {
	token := s.token()
	data, err := json.Marshal(&simpleRequest{
		ClientToken: token,
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{}, 1)
	s.mu.Lock()
	s.chResps[token] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.chResps, token)
		s.mu.Unlock()
	}()

	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("get"),
		QoS:     mqtt.QoS1,
		Payload: []byte(data),
	}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		switch r := res.(type) {
		case *ThingDocument:
			setClientToken(r, "")
			return r, nil
		case *ErrorResponse:
			return nil, r
		default:
			return nil, ErrInvalidResponse
		}
	}
}

func (s *shadow) Delete(ctx context.Context) error {
	token := s.token()
	data, err := json.Marshal(&simpleRequest{
		ClientToken: token,
	})
	if err != nil {
		return err
	}

	ch := make(chan interface{}, 1)
	s.mu.Lock()
	s.chResps[token] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.chResps, token)
		s.mu.Unlock()
	}()

	if err := s.cli.Publish(ctx, &mqtt.Message{
		Topic:   s.topic("delete"),
		QoS:     mqtt.QoS1,
		Payload: []byte(data),
	}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		switch r := res.(type) {
		case *thingDocumentRaw:
			return nil
		case *ErrorResponse:
			return r
		default:
			return ErrInvalidResponse
		}
	}
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
