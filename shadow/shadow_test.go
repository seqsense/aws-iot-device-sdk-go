package shadow

import (
	"reflect"
	"testing"

	mqtt "github.com/at-wat/mqtt-go"
)

func TestShadowHandler(t *testing.T) {
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
