// Copyright 2021 SEQSENSE, Inc.
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
	"reflect"
	"testing"
)

func TestNestedMetadata(t *testing.T) {
	const input = `{"attr1":{"timestamp":1234},"attr2":{"attr3":{"timestamp":1235}},"attr4":[{"timestamp":1236},{"timestamp":1237}]}`
	var expected = NestedMetadata{
		"attr1": Metadata{Timestamp: 1234},
		"attr2": NestedMetadata{
			"attr3": Metadata{Timestamp: 1235},
		},
		"attr4": []interface{}{
			Metadata{Timestamp: 1236},
			Metadata{Timestamp: 1237},
		},
	}

	var v NestedMetadata
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, v) {
		t.Errorf("Expected:\n(%T) %+v\ngot:\n(%T) %+v", expected, expected, v, v)
	}
}
