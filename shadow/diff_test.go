// Copyright 2020-2021 SEQSENSE, Inc.
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
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type testStruct struct {
	A, B, C int
	S       string
}
type testSubStruct struct {
	S1, S2 int
}
type testStructNested struct {
	A, B, C int
	S       testSubStruct
}

func TestStateDiff(t *testing.T) {
	testCases := map[string]struct {
		base    interface{}
		input   interface{}
		diff    interface{}
		hasDiff bool
		err     error
	}{
		"Map2Map": {
			base:    map[string]int{"a": 1, "b": 2, "c": 3},
			input:   map[string]int{"a": 1, "b": 3, "d": 4},
			diff:    map[string]int{"b": 3, "d": 4},
			hasDiff: true,
		},
		"Map2Map_Equal": {
			base:    map[string]int{"a": 1, "b": 2, "c": 3},
			input:   map[string]int{"a": 1, "c": 3, "b": 2},
			hasDiff: false,
		},
		"Map2Map_Remove": {
			base:    map[string]interface{}{"A": 1, "B": 2, "S": "test"},
			input:   map[string]interface{}{"B": 2, "A": 1, "S": nil},
			diff:    map[string]interface{}{"S": nil},
			hasDiff: true,
		},
		"Map2Nil": {
			base:    map[string]int{"a": 1, "b": 2, "c": 3},
			input:   nil,
			diff:    nil,
			hasDiff: true,
		},
		"Nil2Nil": {
			base:    nil,
			input:   nil,
			diff:    nil,
			hasDiff: false,
		},
		"Nil2Struct": {
			base:    nil,
			input:   testStruct{A: 1, B: 3, S: "test"},
			diff:    testStruct{A: 1, B: 3, S: "test"},
			hasDiff: true,
		},
		"Map2Struct": {
			base:    map[string]interface{}{"A": 1, "B": 2, "S": "test"},
			input:   testStruct{A: 1, B: 3, S: "test"},
			diff:    map[string]interface{}{"B": 3, "C": 0},
			hasDiff: true,
		},
		"Map2StructPtr": {
			base:    map[string]interface{}{"A": 1, "B": 2, "S": "test"},
			input:   &testStruct{A: 1, B: 3, S: "test"},
			diff:    map[string]interface{}{"B": 3, "C": 0},
			hasDiff: true,
		},
		"Map2NestedStruct_TypeChange": {
			base:    map[string]interface{}{"A": 1, "B": 2, "S": 3},
			input:   testStructNested{A: 1, B: 3, S: testSubStruct{S1: 2}},
			diff:    map[string]interface{}{"B": 3, "C": 0, "S": testSubStruct{S1: 2}},
			hasDiff: true,
		},
		"Map2Struct_Equal": {
			base:    map[string]interface{}{"B": 2, "A": 1, "C": 0, "S": "test"},
			input:   testStruct{A: 1, B: 2, S: "test"},
			hasDiff: false,
		},
		"Map2StructPtr_Equal": {
			base:    map[string]interface{}{"B": 2, "A": 1, "C": 0, "S": "test"},
			input:   &testStruct{A: 1, B: 2, S: "test"},
			hasDiff: false,
		},
		"Nil2Map": {
			base:    nil,
			input:   map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			diff:    map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			hasDiff: true,
		},
		"WrongType": {
			base:  map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			input: map[int]interface{}{1: 1, 3: 3, 4: "test"},
			err:   ErrUnsupportedMapKeyType,
		},
		"WrongType2": {
			base:  map[int]interface{}{1: 1, 3: 3, 4: "test"},
			input: map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			err:   ErrUnsupportedMapKeyType,
		},
		"InterfaceSlice": {
			base: map[string]interface{}{"A": []interface{}{1, 2, 4, 5}},
			input: struct{ A []int }{
				A: []int{1, 2, 3, 4, 5},
			},
			diff:    map[string]interface{}{"A": []interface{}{1, 2, 3, 4, 5}},
			hasDiff: true,
		},
		"InterfaceSliceEqual": {
			base: map[string]interface{}{"A": []interface{}{1, 2, 3, 4, 5}},
			input: struct{ A []int }{
				A: []int{1, 2, 3, 4, 5},
			},
			hasDiff: false,
		},
		"InterfaceArray": {
			base: map[string]interface{}{"A": []interface{}{1, 2, 4, 5}},
			input: struct{ A [5]int }{
				A: [5]int{1, 2, 3, 4, 5},
			},
			diff:    map[string]interface{}{"A": []interface{}{1, 2, 3, 4, 5}},
			hasDiff: true,
		},
		"InterfaceArrayEqual": {
			base: map[string]interface{}{"A": []interface{}{1, 2, 3, 4, 5}},
			input: struct{ A [5]int }{
				A: [5]int{1, 2, 3, 4, 5},
			},
			hasDiff: false,
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			diff, hasDiff, err := stateDiff(tt.base, tt.input)
			if tt.err != nil {
				if !errors.Is(tt.err, err) {
					t.Errorf("Expected error: '%v', got: '%v'", tt.err, err)
				}
				return
			}
			if hasDiff != tt.hasDiff {
				t.Errorf("Expected: hasDiff=%v, got: hasDiff=%v", tt.hasDiff, hasDiff)
			}
			if a, b := fmt.Sprint(tt.diff), fmt.Sprint(diff); a != b {
				t.Errorf("Expected: %+v, got: %+v", a, b)
			}
		})
	}
}

func TestAttributeKeys(t *testing.T) {
	testCases := map[string]struct {
		input    interface{}
		keys     []string
		hasChild bool
		err      error
	}{
		"Slice": {
			input: []int{1, 2, 3},
		},
		"Map": {
			input:    map[string]int{"a": 1, "b": 2},
			keys:     []string{"a", "b"},
			hasChild: true,
		},
		"Struct": {
			input:    testStruct{},
			keys:     []string{"A", "B", "C", "S"},
			hasChild: true,
		},
		"StructPtr": {
			input:    &testStruct{},
			keys:     []string{"A", "B", "C", "S"},
			hasChild: true,
		},
		"NestedStruct": {
			input:    testStructNested{},
			keys:     []string{"A", "B", "C", "S"},
			hasChild: true,
		},
		"Int": {
			input: 1,
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			keys, hasChild, err := attributeKeys(tt.input)
			if tt.err != nil {
				if !errors.Is(tt.err, err) {
					t.Errorf("Expected error: '%v', got: '%v'", tt.err, err)
				}
				return
			}

			if hasChild != tt.hasChild {
				t.Errorf("Expected: hasChild=%v, got: hasChild=%v", tt.hasChild, hasChild)
			}
			sort.Strings(keys)
			if !reflect.DeepEqual(tt.keys, keys) {
				t.Errorf("Expected: %+v, got: %+v", tt.keys, keys)
			}
		})
	}
}

func TestAttributeByKey(t *testing.T) {
	testCases := map[string]struct {
		input    interface{}
		keyValue map[string]interface{}
		err      error
	}{
		"Map": {
			input:    map[string]int{"a": 1, "b": 2},
			keyValue: map[string]interface{}{"a": 1, "b": 2},
		},
		"Struct": {
			input:    testStruct{A: 2, S: "test"},
			keyValue: map[string]interface{}{"A": 2, "B": 0, "C": 0, "S": "test"},
		},
		"StructPtr": {
			input:    &testStruct{A: 2, S: "test"},
			keyValue: map[string]interface{}{"A": 2, "B": 0, "C": 0, "S": "test"},
		},
		"NestedStruct": {
			input:    testStructNested{A: 3, S: testSubStruct{S2: 4}},
			keyValue: map[string]interface{}{"A": 3, "S": testSubStruct{S2: 4}},
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			for k, v := range tt.keyValue {
				a, err := attributeByKey(tt.input, k)
				if tt.err != nil {
					if !errors.Is(tt.err, err) {
						t.Errorf("Expected error: '%v', got: '%v'", tt.err, err)
					}
					return
				}
				if !reflect.DeepEqual(v, a) {
					t.Errorf("Expected: %+v, got: %+v", v, a)
				}
			}
		})
	}
}

func BenchmarkStateDiff(b *testing.B) {
	testCases := map[string]struct {
		base, input interface{}
	}{
		"Map2Map": {
			base: map[string]interface{}{
				"String":  "test data",
				"Integer": 1234,
				"Array":   []string{"value1", "value2", "value3", "value4"},
			},
			input: map[string]interface{}{
				"String":  "test data2",
				"Integer": 1235,
				"Array":   []string{"value1", "value2", "value3", "value4"},
			},
		},
		"Map2Struct": {
			base: map[string]interface{}{
				"String":  "test data",
				"Integer": 1234,
				"Array":   []string{"value1", "value2", "value3", "value4"},
			},
			input: struct {
				String  string
				Integer int
				Array   []string
			}{
				String:  "test data2",
				Integer: 1235,
				Array:   []string{"value1", "value2", "value3", "value4"},
			},
		},
		"Map2StructPtr": {
			base: map[string]interface{}{
				"String":  "test data",
				"Integer": 1234,
				"Array":   []string{"value1", "value2", "value3", "value4"},
			},
			input: &struct {
				String  string
				Integer int
				Array   []string
			}{
				String:  "test data2",
				Integer: 1235,
				Array:   []string{"value1", "value2", "value3", "value4"},
			},
		},
	}
	for name, tt := range testCases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := stateDiff(tt.base, tt.input)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
