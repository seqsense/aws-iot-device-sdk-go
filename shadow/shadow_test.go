package shadow

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	mqtt "github.com/at-wat/mqtt-go"
)

func TestHandlers(t *testing.T) {
	t.Run("Rejected", func(t *testing.T) {
		chErr := make(chan error, 1)
		s := &shadow{
			doc: &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{"key": "value_init"},
				},
				Version:   2,
				Timestamp: 12345,
			},
			onDelta: func(delta map[string]interface{}) {
				t.Error("onDelta must not be called on rejected")
			},
			onError: func(err error) {
				chErr <- err
			},
		}
		expectedDoc := &ThingDocument{
			State: ThingState{
				Desired: map[string]interface{}{"key": "value_init"},
			},
			Version:   2,
			Timestamp: 12345,
		}

		s.rejected(&mqtt.Message{})
		select {
		case err := <-chErr:
			if err == nil {
				t.Error("onError must be called with non-nil error")
			}
		default:
			t.Fatal("Timeout")
		}
		if !reflect.DeepEqual(*expectedDoc, *s.doc) {
			t.Error("Document must not be changed on reject")
		}
	})

	t.Run("Accepted", func(t *testing.T) {
		t.Run("Get", func(t *testing.T) {
			chErr := make(chan error, 1)
			chDelta := make(chan map[string]interface{}, 1)
			s := &shadow{
				doc: &ThingDocument{
					State: ThingState{
						Desired: map[string]interface{}{"key": "value_init"},
					},
					Version:   2,
					Timestamp: 12345,
				},
				onDelta: func(delta map[string]interface{}) {
					chDelta <- delta
				},
				onError: func(err error) {
					chErr <- err
				},
			}
			expectedDoc := &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{"key2": "value2"},
					Delta:   map[string]interface{}{"key2": "value2"},
				},
				Version:   3,
				Timestamp: 12346,
			}
			s.getAccepted(&mqtt.Message{Payload: []byte(
				"{" +
					"  \"state\": {" +
					"    \"desired\": {\"key2\": \"value2\"}," +
					"    \"delta\": {\"key2\": \"value2\"}" +
					"  }," +
					"  \"version\": 3," +
					"  \"timestamp\": 12346" +
					"}",
			)})

			select {
			case err := <-chErr:
				t.Error(err)
			case delta := <-chDelta:
				if !reflect.DeepEqual(expectedDoc.State.Delta, delta) {
					t.Errorf("Expected delta: %v, got: %v",
						expectedDoc.State.Delta, delta,
					)
				}
			default:
				t.Fatal("Timeout")
			}
			if !reflect.DeepEqual(expectedDoc, s.doc) {
				t.Errorf("Expected state: %v, got: %v",
					expectedDoc, s.doc,
				)
			}
		})
		t.Run("Delete", func(t *testing.T) {
			chErr := make(chan error, 1)
			s := &shadow{
				doc: &ThingDocument{
					State: ThingState{
						Desired: map[string]interface{}{"key": "value_init"},
					},
					Version:   2,
					Timestamp: 12345,
				},
				onDelta: func(delta map[string]interface{}) {
					t.Error("Delete must not trigger onDelta")
				},
				onError: func(err error) {
					chErr <- err
				},
			}
			s.deleteAccepted(&mqtt.Message{Payload: []byte("{}")})

			select {
			case err := <-chErr:
				t.Error(err)
			default:
			}
			if s.doc != nil {
				t.Errorf("Document must be nil after delete")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("Delta", func(t *testing.T) {
			chErr := make(chan error, 1)
			chDelta := make(chan map[string]interface{}, 1)
			s := &shadow{
				doc: &ThingDocument{
					State: ThingState{
						Desired: map[string]interface{}{"key2": "value2"},
					},
					Version:   2,
					Timestamp: 12345,
				},
				onDelta: func(delta map[string]interface{}) {
					chDelta <- delta
				},
				onError: func(err error) {
					chErr <- err
				},
			}
			expectedDoc := &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{"key2": "value2"},
					Delta:   map[string]interface{}{"key2": "value2"},
				},
				Version:   3,
				Timestamp: 12346,
			}
			s.updateDelta(&mqtt.Message{Payload: []byte(
				"{" +
					"  \"state\": {" +
					"    \"key2\": \"value2\"" +
					"  }," +
					"  \"version\": 3," +
					"  \"timestamp\": 12346" +
					"}",
			)})

			select {
			case err := <-chErr:
				t.Error(err)
			case delta := <-chDelta:
				if !reflect.DeepEqual(expectedDoc.State.Delta, delta) {
					t.Errorf("Expected delta: %v, got: %v",
						expectedDoc.State.Delta, delta,
					)
				}
			default:
				t.Fatal("Timeout")
			}
			if !reflect.DeepEqual(expectedDoc, s.doc) {
				t.Errorf("Expected state: %v, got: %v",
					expectedDoc, s.doc,
				)
			}
		})
		t.Run("Accepted", func(t *testing.T) {
			chErr := make(chan error, 1)
			chDelta := make(chan map[string]interface{}, 1)
			s := &shadow{
				doc: &ThingDocument{
					State: ThingState{
						Reported: map[string]interface{}{"key1": "value1"},
					},
					Version:   2,
					Timestamp: 12345,
				},
				onDelta: func(delta map[string]interface{}) {
					chDelta <- delta
				},
				onError: func(err error) {
					chErr <- err
				},
			}
			expectedDoc := &ThingDocument{
				State: ThingState{
					Reported: map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
				},
				Version:   3,
				Timestamp: 12346,
			}
			s.updateAccepted(&mqtt.Message{Payload: []byte(
				"{" +
					"  \"state\": {" +
					"    \"Reported\": {" +
					"      \"key2\": \"value2\"" +
					"    }" +
					"  }," +
					"  \"version\": 3," +
					"  \"timestamp\": 12346" +
					"}",
			)})

			select {
			case err := <-chErr:
				t.Error(err)
			case <-chDelta:
				t.Error("cbDelta must not be called on update accept")
			default:
			}
			if !reflect.DeepEqual(expectedDoc, s.doc) {
				t.Errorf("Expected state: %v, got: %v",
					expectedDoc, s.doc,
				)
			}
		})
		t.Run("OldDelta", func(t *testing.T) {
			s := &shadow{
				doc: &ThingDocument{
					State: ThingState{
						Desired: map[string]interface{}{"key2": "value2"},
					},
					Version:   2,
					Timestamp: 12345,
				},
				onDelta: func(delta map[string]interface{}) {
					t.Error("Old delta must be discarded")
				},
				onError: func(err error) {
					t.Error("Old delta must be silently discarded")
				},
			}
			expectedDoc := &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{"key2": "value2"},
				},
				Version:   2,
				Timestamp: 12345,
			}
			s.updateDelta(&mqtt.Message{Payload: []byte(
				"{" +
					"  \"state\": {" +
					"    \"key2\": \"value\"" +
					"  }," +
					"  \"version\": 1," +
					"  \"timestamp\": 12343" +
					"}",
			)})

			if !reflect.DeepEqual(expectedDoc, s.doc) {
				t.Errorf("Expected state: %v, got: %v",
					expectedDoc, s.doc,
				)
			}
		})
	})
}

func TestOnDelta(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	j, err := New(ctx, &dummyClient{})
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{
		"key": "value",
	}

	done := make(chan struct{})
	j.OnDelta(func(delta map[string]interface{}) {
		if !reflect.DeepEqual(expected, delta) {
			t.Fatalf("Expected delta: %v, got: %v", expected, delta)
		}
		close(done)
	})

	req := &thingDelta{
		State: map[string]interface{}{
			"key": "value",
		},
		Version: 10,
	}
	breq, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	j.Serve(&mqtt.Message{
		Topic:   j.(*shadow).topic("update/delta"),
		Payload: breq,
	})

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("Timeout")
	}
}

func TestGet(t *testing.T) {
	testCases := map[string]struct {
		response      interface{}
		responseTopic string
		expected      interface{}
		err           error
	}{
		"Success": {
			response: &ThingDocument{
				Version: 5,
				State: ThingState{
					Desired:  map[string]interface{}{"key": "value"},
					Reported: map[string]interface{}{"key": "value"},
				},
			},
			responseTopic: "get/accepted",
			expected: &ThingDocument{
				Version: 5,
				State: ThingState{
					Desired:  map[string]interface{}{"key": "value"},
					Reported: map[string]interface{}{"key": "value"},
				},
			},
		},
		"Error": {
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "get/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &simpleRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, req.ClientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					s.Serve(&mqtt.Message{
						Topic:   s.(*shadow).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := s.Get(ctx)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			} else {
				if !reflect.DeepEqual(testCase.expected, doc) {
					t.Errorf("Expected document: %v, got: %v", testCase.expected, doc)
				}
			}
		})
	}
}

func TestDesire(t *testing.T) {
	testCases := map[string]struct {
		input         map[string]interface{}
		response      interface{}
		responseTopic string
		expected      interface{}
		err           error
	}{
		"Success": {
			input: map[string]interface{}{"key": "value"},
			response: &ThingDocument{
				Version: 5,
				State: ThingState{
					Desired: map[string]interface{}{"key": "value"},
				},
			},
			responseTopic: "update/accepted",
			expected: &ThingDocument{
				Version: 5,
				State: ThingState{
					Desired:  map[string]interface{}{"key": "value"},
					Reported: map[string]interface{}{},
					Delta:    map[string]interface{}{},
				},
			},
		},
		"Error": {
			input: map[string]interface{}{"key": "value"},
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "update/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &simpleRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, req.ClientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					s.Serve(&mqtt.Message{
						Topic:   s.(*shadow).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := s.Desire(ctx, testCase.input)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			} else {
				if !reflect.DeepEqual(testCase.expected, doc) {
					t.Errorf("Expected document: %v, got: %v", testCase.expected, doc)
				}
			}
		})
	}
}

func TestReport(t *testing.T) {
	testCases := map[string]struct {
		input         map[string]interface{}
		response      interface{}
		responseTopic string
		expected      interface{}
		err           error
	}{
		"Success": {
			input: map[string]interface{}{"key": "value"},
			response: &ThingDocument{
				Version: 5,
				State: ThingState{
					Reported: map[string]interface{}{"key": "value"},
				},
			},
			responseTopic: "update/accepted",
			expected: &ThingDocument{
				Version: 5,
				State: ThingState{
					Reported: map[string]interface{}{"key": "value"},
					Desired:  map[string]interface{}{},
					Delta:    map[string]interface{}{},
				},
			},
		},
		"Error": {
			input: map[string]interface{}{"key": "value"},
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "update/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &simpleRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, req.ClientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					s.Serve(&mqtt.Message{
						Topic:   s.(*shadow).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := s.Report(ctx, testCase.input)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			} else {
				if !reflect.DeepEqual(testCase.expected, doc) {
					t.Errorf("Expected document: %v, got: %v", testCase.expected, doc)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testCases := map[string]struct {
		response      interface{}
		responseTopic string
		err           error
	}{
		"Success": {
			response:      &ThingDocument{Version: 10},
			responseTopic: "delete/accepted",
		},
		"Error": {
			response: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
			responseTopic: "delete/rejected",
			err: &ErrorResponse{
				Code:    "Failed",
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			cli := &dummyClient{
				publish: func(ctx context.Context, msg *mqtt.Message) {
					req := &simpleRequest{}
					if err := json.Unmarshal(msg.Payload, req); err != nil {
						t.Error(err)
						cancel()
						return
					}
					res := testCase.response
					setClientToken(res, req.ClientToken)
					bres, err := json.Marshal(res)
					if err != nil {
						t.Error(err)
						cancel()
						return
					}
					s.Serve(&mqtt.Message{
						Topic:   s.(*shadow).topic(testCase.responseTopic),
						Payload: bres,
					})
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}

			err = s.Delete(ctx)
			if err != nil {
				setClientToken(err, "")
				if !reflect.DeepEqual(testCase.err, err) {
					t.Fatalf("Expected error: %v, got: %v", testCase.err, err)
				}
			}
		})
	}
}

type dummyClient struct {
	publish func(context.Context, *mqtt.Message)
	handler mqtt.Handler
}

func (c *dummyClient) ThingName() string {
	return "test"
}

func (*dummyClient) Connect(ctx context.Context, clientID string, opts ...mqtt.ConnectOption) (sessionPresent bool, err error) {
	return false, nil
}
func (*dummyClient) Disconnect(ctx context.Context) error {
	panic("not implemented")
}
func (c *dummyClient) Publish(ctx context.Context, message *mqtt.Message) error {
	c.publish(ctx, message)
	return nil
}
func (*dummyClient) Subscribe(ctx context.Context, subs ...mqtt.Subscription) error {
	return nil
}
func (*dummyClient) Unsubscribe(ctx context.Context, subs ...string) error {
	panic("not implemented")
}
func (*dummyClient) Ping(ctx context.Context) error {
	panic("not implemented")
}
func (c *dummyClient) Handle(handler mqtt.Handler) {
	c.handler = handler
}
