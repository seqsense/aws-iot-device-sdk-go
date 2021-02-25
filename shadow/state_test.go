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
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

func TestThingDocument_update(t *testing.T) {
	doc := &ThingDocument{
		State:     ThingState{Desired: NestedState{"key": "value_init"}},
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
		State:     ThingState{Desired: NestedState{"key": "value_init"}},
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
		State:     ThingState{Desired: NestedState{"key": "new value"}},
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
		State:     ThingState{Desired: NestedState{}},
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
			Desired: NestedState{"key": NestedState{"key": "value"}},
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

func TestNestedState_MapTo(t *testing.T) {
	testCases := map[string]struct {
		input     NestedState
		container interface{}
		expected  interface{}
	}{
		"OneByOne": {
			input: NestedState{
				"A": NestedState{
					"B": 1,
				},
				"C": 2,
			},
			container: &struct {
				A struct{ B int }
				C int
			}{},
			expected: &struct {
				A struct{ B int }
				C int
			}{
				A: struct{ B int }{
					B: 1,
				},
				C: 2,
			},
		},
		"ExtraField": {
			input: NestedState{
				"A": NestedState{
					"B": 1,
				},
				"C": 2,
				"D": 2,
			},
			container: &struct {
				A struct{ B int }
				C int
			}{},
			expected: &struct {
				A struct{ B int }
				C int
			}{
				A: struct{ B int }{
					B: 1,
				},
				C: 2,
			},
		},
		"MissingField": {
			input: NestedState{
				"A": NestedState{
					"B": 1,
				},
			},
			container: &struct {
				A struct{ B int }
				C int
			}{},
			expected: &struct {
				A struct{ B int }
				C int
			}{
				A: struct{ B int }{
					B: 1,
				},
				C: 0,
			},
		},
		"SliceConvert": {
			input: NestedState{
				"A": []interface{}{1.0, 2.0, 3, 4.0, 5},
			},
			container: &struct {
				A []int
			}{},
			expected: &struct {
				A []int
			}{
				A: []int{1, 2, 3, 4, 5},
			},
		},
		"Slice": {
			input: NestedState{
				"A": []interface{}{1, 2, 3, 4, 5},
			},
			container: &struct {
				A []int
			}{},
			expected: &struct {
				A []int
			}{
				A: []int{1, 2, 3, 4, 5},
			},
		},
		"StructSlice": {
			input: NestedState{
				"A": []interface{}{
					NestedState{"B": 1},
					NestedState{"B": 2},
				},
			},
			container: &struct {
				A []struct{ B int }
			}{},
			expected: &struct {
				A []struct{ B int }
			}{
				A: []struct{ B int }{
					{B: 1},
					{B: 2},
				},
			},
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			err := tt.input.MapTo(&tt.container)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.expected, tt.container) {
				t.Errorf("Expected:\n%v\ngot:\n%v", tt.expected, tt.container)
			}
		})
	}
}
