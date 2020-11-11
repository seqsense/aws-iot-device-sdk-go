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

package shadow

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/at-wat/mqtt-go"
	mockmqtt "github.com/at-wat/mqtt-go/mock"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

type mockDevice struct {
	*mockmqtt.Client
}

func (d *mockDevice) ThingName() string {
	return "test"
}

func TestNew(t *testing.T) {
	errDummy := errors.New("dummy error")

	t.Run("SubscribeError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		cli := &mockDevice{&mockmqtt.Client{
			SubscribeFn: func(ctx context.Context, subs ...mqtt.Subscription) error {
				return errDummy
			},
		}}
		_, err := New(ctx, cli)
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		if !errors.Is(err, errDummy) {
			t.Errorf("Expected error: %v, got: %v", errDummy, err)
		}
	})
}

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
			var ie *ioterr.Error
			if !errors.As(err, &ie) {
				t.Errorf("Expected error type: %T, got: %T", ie, err)
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

func TestHandlers_InvalidResponse(t *testing.T) {
	for _, topic := range []string{
		"update/delta",
		"update/accepted",
		"update/rejected",
		"get/accepted",
		"get/rejected",
		"delete/accepted",
		"delete/rejected",
	} {
		topic := topic
		t.Run(topic, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			var cli *mockDevice
			cli = &mockDevice{Client: &mockmqtt.Client{}}

			s, err := New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}
			chErr := make(chan error, 1)
			s.OnError(func(err error) { chErr <- err })
			cli.Handle(s)

			cli.Serve(&mqtt.Message{
				Topic:   s.(*shadow).topic(topic),
				Payload: []byte{0xff, 0xff, 0xff},
			})

			select {
			case err := <-chErr:
				var ie *ioterr.Error
				if !errors.As(err, &ie) {
					t.Errorf("Expected error type: %T, got: %T", ie, err)
				}
			case <-ctx.Done():
				t.Fatal("Timeout")
			}
		})
	}
}

func TestOnDelta(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cli := &mockDevice{&mockmqtt.Client{}}
	s, err := New(ctx, cli)
	if err != nil {
		t.Fatal(err)
	}
	cli.Handle(s)

	expected := map[string]interface{}{
		"key": "value",
	}

	done := make(chan struct{})
	s.OnDelta(func(delta map[string]interface{}) {
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
	cli.Serve(&mqtt.Message{
		Topic:   s.(*shadow).topic("update/delta"),
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
				Code:    400,
				Message: "Reason",
			},
			responseTopic: "get/rejected",
			err: &ErrorResponse{
				Code:    400,
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			var cli *mockDevice
			cli = &mockDevice{
				Client: &mockmqtt.Client{
					PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
						req := &simpleRequest{}
						if err := json.Unmarshal(msg.Payload, req); err != nil {
							t.Error(err)
							cancel()
							return err
						}
						res := testCase.response
						setClientToken(res, req.ClientToken)
						bres, err := json.Marshal(res)
						if err != nil {
							t.Error(err)
							cancel()
							return err
						}
						cli.Serve(&mqtt.Message{
							Topic:   s.(*shadow).topic(testCase.responseTopic),
							Payload: bres,
						})
						return nil
					},
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}
			cli.Handle(s)

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
				Code:    400,
				Message: "Reason",
			},
			responseTopic: "update/rejected",
			err: &ErrorResponse{
				Code:    400,
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			var cli *mockDevice
			cli = &mockDevice{
				Client: &mockmqtt.Client{
					PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
						req := &simpleRequest{}
						if err := json.Unmarshal(msg.Payload, req); err != nil {
							t.Error(err)
							cancel()
							return err
						}
						res := testCase.response
						setClientToken(res, req.ClientToken)
						bres, err := json.Marshal(res)
						if err != nil {
							t.Error(err)
							cancel()
							return err
						}
						cli.Serve(&mqtt.Message{
							Topic:   s.(*shadow).topic(testCase.responseTopic),
							Payload: bres,
						})
						return nil
					},
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}
			cli.Handle(s)

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
				Code:    400,
				Message: "Reason",
			},
			responseTopic: "update/rejected",
			err: &ErrorResponse{
				Code:    400,
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			var cli *mockDevice
			cli = &mockDevice{
				Client: &mockmqtt.Client{
					PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
						req := &simpleRequest{}
						if err := json.Unmarshal(msg.Payload, req); err != nil {
							t.Error(err)
							cancel()
							return err
						}
						res := testCase.response
						setClientToken(res, req.ClientToken)
						bres, err := json.Marshal(res)
						if err != nil {
							t.Error(err)
							cancel()
							return err
						}
						cli.Serve(&mqtt.Message{
							Topic:   s.(*shadow).topic(testCase.responseTopic),
							Payload: bres,
						})
						return nil
					},
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}
			cli.Handle(s)

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
				Code:    400,
				Message: "Reason",
			},
			responseTopic: "delete/rejected",
			err: &ErrorResponse{
				Code:    400,
				Message: "Reason",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)

			var s Shadow
			var cli *mockDevice
			cli = &mockDevice{
				Client: &mockmqtt.Client{
					PublishFn: func(ctx context.Context, msg *mqtt.Message) error {
						req := &simpleRequest{}
						if err := json.Unmarshal(msg.Payload, req); err != nil {
							t.Error(err)
							cancel()
							return err
						}
						res := testCase.response
						setClientToken(res, req.ClientToken)
						bres, err := json.Marshal(res)
						if err != nil {
							t.Error(err)
							cancel()
							return err
						}
						s.Serve(&mqtt.Message{
							Topic:   s.(*shadow).topic(testCase.responseTopic),
							Payload: bres,
						})
						return nil
					},
				},
			}
			var err error
			s, err = New(ctx, cli)
			if err != nil {
				t.Fatal(err)
			}
			cli.Handle(s)

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
