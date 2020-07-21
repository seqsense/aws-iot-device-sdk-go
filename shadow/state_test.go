package shadow

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
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
