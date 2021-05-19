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
	"regexp"
)

var regexLeafJSON = regexp.MustCompile(`^{[^{}]*}$`)

// ThingStateMetadata represents Thing Shadow State metadata.
type ThingStateMetadata struct {
	Desired  NestedMetadata `json:"desired,omitempty"`
	Reported NestedMetadata `json:"reported,omitempty"`
	Delta    NestedMetadata `json:"delta,omitempty"`
}

// Metadata represents metadata of state attribute.
type Metadata struct {
	Timestamp int `json:"timestamp"`
}

// NestedMetadata is JSON unmarshaller for state metadata.
type NestedMetadata map[string]interface{}

// UnmarshalJSON implements json.Unmarshaler.
func (n *NestedMetadata) UnmarshalJSON(b []byte) error {
	if *n == nil {
		*n = make(map[string]interface{})
	}
	j := make(map[string]json.RawMessage)
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	for k, v := range j {
		v2, err := unmarshalMetadataImpl(v)
		if err != nil {
			return err
		}
		(*n)[k] = v2
	}
	return nil
}

func unmarshalMetadataImpl(b []byte) (interface{}, error) {
	switch {
	case b[0] == '[' && b[len(b)-1] == ']':
		var v2 []json.RawMessage
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		var v5 []interface{}
		for _, v3 := range v2 {
			v4, err := unmarshalMetadataImpl(v3)
			if err != nil {
				return nil, err
			}
			v5 = append(v5, v4)
		}
		return v5, nil
	case !regexLeafJSON.Match(b):
		var v2 NestedMetadata
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		return v2, nil
	default:
		var v2 Metadata
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		return v2, nil
	}
}
