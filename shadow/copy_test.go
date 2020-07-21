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
	"bytes"
	"encoding/json"
	"testing"
)

func TestClone(t *testing.T) {
	testCases := map[string]struct {
		doc    *ThingDocument
		mutate func(*ThingDocument)
	}{
		"Base": {
			doc: &ThingDocument{Version: 10},
			mutate: func(doc *ThingDocument) {
				doc.Version = 1
			},
		},
		"Map": {
			doc: &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{
						"key": "value",
					},
				},
			},
			mutate: func(doc *ThingDocument) {
				doc.State.Desired["key"] = "value2"
			},
		},
		"MapInMap": {
			doc: &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{
						"key": map[string]interface{}{
							"childKey": "value",
						},
					},
				},
			},
			mutate: func(doc *ThingDocument) {
				doc.State.Desired["key"].(map[string]interface{})["childKey"] = "value2"
			},
		},
		"SliceInMap": {
			doc: &ThingDocument{
				State: ThingState{
					Desired: map[string]interface{}{
						"key": []interface{}{
							"value1",
							"value2",
						},
					},
				},
			},
			mutate: func(doc *ThingDocument) {
				doc.State.Desired["key"].([]interface{})[0] = "value1b"
			},
		},
		"Nil": {
			doc:    nil,
			mutate: func(doc *ThingDocument) {},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			originalJson, err := json.Marshal(testCase.doc)
			if err != nil {
				t.Fatal(err)
			}
			cloned := testCase.doc.clone()
			testCase.mutate(cloned)

			resultJson, err := json.Marshal(testCase.doc)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(originalJson, resultJson) {
				t.Errorf(
					"Original document is mutated by updating cloned one\nexpected: %s\ngot: %s",
					originalJson, resultJson,
				)
			}
		})
	}
}
