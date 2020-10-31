package shadow

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/internal/ioterr"
)

func TestThingDocument_update(t *testing.T) {
	doc := &ThingDocument{
		State:     ThingState{Desired: map[string]interface{}{"key": "value_init"}},
		Version:   2,
		Timestamp: 12345,
	}

	if err := doc.update(&thingDocumentRaw{
		State:     thingStateRaw{Desired: json.RawMessage(`{"key": "ignored"}`)},
		Version:   1,
		Timestamp: 0,
	}); err != nil {
		t.Fatal(err)
	}

	expected0 := &ThingDocument{
		State:     ThingState{Desired: map[string]interface{}{"key": "value_init"}},
		Version:   2,
		Timestamp: 12345,
	}
	if !reflect.DeepEqual(*expected0, *doc) {
		t.Errorf(
			"Old version must be discarded\nexpected: %v\ngot: %v",
			*expected0, *doc,
		)
	}

	if err := doc.update(&thingDocumentRaw{
		State:     thingStateRaw{Desired: json.RawMessage(`{"key": "new value"}`)},
		Version:   4,
		Timestamp: 12346,
	}); err != nil {
		t.Fatal(err)
	}

	expected1 := &ThingDocument{
		State:     ThingState{Desired: map[string]interface{}{"key": "new value"}},
		Version:   4,
		Timestamp: 12346,
	}
	if !reflect.DeepEqual(*expected1, *doc) {
		t.Errorf(
			"Document must be update by new version\nexpected: %v\ngot: %v",
			*expected1, *doc,
		)
	}

	if err := doc.update(&thingDocumentRaw{
		State:     thingStateRaw{Desired: json.RawMessage(`null`)},
		Version:   5,
		Timestamp: 12347,
	}); err != nil {
		t.Fatal(err)
	}

	expected2 := &ThingDocument{
		State:     ThingState{Desired: map[string]interface{}{}},
		Version:   5,
		Timestamp: 12347,
	}
	if !reflect.DeepEqual(*expected2, *doc) {
		t.Errorf(
			"Document must be cleared by nil\nexpected: %v\ngot: %v",
			*expected2, *doc,
		)
	}

	if err := doc.update(&thingDocumentRaw{
		State:     thingStateRaw{Desired: json.RawMessage(`{"key": {"key": "value"}}`)},
		Version:   6,
		Timestamp: 12348,
	}); err != nil {
		t.Fatal(err)
	}

	expected3 := &ThingDocument{
		State: ThingState{
			Desired: map[string]interface{}{"key": map[string]interface{}{"key": "value"}},
		},
		Version:   6,
		Timestamp: 12348,
	}
	if !reflect.DeepEqual(*expected3, *doc) {
		t.Errorf(
			"Document must be cleared by nil\nexpected: %v\ngot: %v",
			*expected3, *doc,
		)
	}

	t.Run("InvalidState", func(t *testing.T) {
		err := doc.update(&thingDocumentRaw{
			State:     thingStateRaw{Desired: json.RawMessage(`{"broken"}`)},
			Version:   10,
			Timestamp: 20000,
		})
		var ie *ioterr.Error
		if !errors.As(err, &ie) {
			t.Errorf("Expected error type: %T, got: %T", ie, err)
		}
		var je *json.SyntaxError
		if !errors.As(err, &je) {
			t.Errorf("Expected error type: %T, got: %T", je, err)
		}
	})
}

func TestErrorResponse(t *testing.T) {
	err := &ErrorResponse{
		Code:    100,
		Message: "error message",
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "100") {
		t.Error("Error string should contain error code")
	}
	if !strings.Contains(errStr, "error message") {
		t.Error("Error string should contain error message")
	}
}
