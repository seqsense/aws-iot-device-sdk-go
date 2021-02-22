package shadow

import (
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
		"Nil2Map": {
			base:    nil,
			input:   map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			diff:    map[string]interface{}{"A": 1, "B": 3, "S": "test"},
			hasDiff: true,
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			diff, hasDiff := stateDiff(tt.base, tt.input)
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
			input: struct {
				A, B, C int
				S       string
			}{},
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
			keys, hasChild := attributeKeys(tt.input)
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
	}{
		"Map": {
			input:    map[string]int{"a": 1, "b": 2},
			keyValue: map[string]interface{}{"a": 1, "b": 2},
		},
		"Struct": {
			input:    testStruct{A: 2, S: "test"},
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
				a, _ := attributeByKey(tt.input, k)
				if !reflect.DeepEqual(v, a) {
					t.Errorf("Expected: %+v, got: %+v", v, a)
				}
			}
		})
	}
}
