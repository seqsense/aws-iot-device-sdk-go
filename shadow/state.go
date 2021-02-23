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
	"fmt"

	"github.com/seqsense/aws-iot-device-sdk-go/v5/internal/ioterr"
)

// ErrVersionConflict means thing state update was aborted due to version conflict.
var ErrVersionConflict = errors.New("version conflict")

type simpleRequest struct {
	ClientToken string `json:"clientToken"`
}

// ErrorResponse represents error response from AWS IoT.
type ErrorResponse struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Timestamp   int64  `json:"timestamp"`
	ClientToken string `json:"clientToken"`
}

// Error implements error interface.
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%d (%s): %s", e.Code, e.ClientToken, e.Message)
}

// ThingState represents Thing Shadow State.
type ThingState struct {
	Desired  NestedState `json:"desired,omitempty"`
	Reported NestedState `json:"reported,omitempty"`
	Delta    NestedState `json:"delta,omitempty"`
}

// ThingDocument represents Thing Shadow Document.
type ThingDocument struct {
	State       ThingState         `json:"state"`
	Metadata    ThingStateMetadata `json:"metadata"`
	Version     int                `json:"version,omitempty"`
	Timestamp   int                `json:"timestamp,omitempty"`
	ClientToken string             `json:"clientToken,omitempty"`
}

type thingStateRaw struct {
	Desired     json.RawMessage `json:"desired,omitempty"`
	Reported    json.RawMessage `json:"reported,omitempty"`
	Delta       json.RawMessage `json:"delta,omitempty"`
	ClientToken string          `json:"clientToken,omitempty"`
}

type thingDocumentRaw struct {
	State       thingStateRaw `json:"state"`
	Metadata    thingStateRaw `json:"metadata"`
	Version     int           `json:"version,omitempty"`
	Timestamp   int           `json:"timestamp,omitempty"`
	ClientToken string        `json:"clientToken,omitempty"`
}

type thingDelta struct {
	State     NestedState    `json:"state"`
	Metadata  NestedMetadata `json:"metadata"`
	Version   int            `json:"version,omitempty"`
	Timestamp int            `json:"timestamp,omitempty"`
}

func (s *ThingDocument) update(state *thingDocumentRaw) error {
	if s.Version > state.Version {
		// Received an old version; just ignore it.
		return nil
	}
	s.Version = state.Version
	s.Timestamp = state.Timestamp
	if err := updateStateRaw(s.State.Desired, state.State.Desired); err != nil {
		return ioterr.New(err, "updating desired state")
	}
	if err := updateStateMetadataRaw(s.Metadata.Desired, state.Metadata.Desired); err != nil {
		return ioterr.New(err, "updating desired state")
	}
	if err := updateStateRaw(s.State.Reported, state.State.Reported); err != nil {
		return ioterr.New(err, "updating reported state")
	}
	if err := updateStateMetadataRaw(s.Metadata.Reported, state.Metadata.Reported); err != nil {
		return ioterr.New(err, "updating reported state")
	}
	return nil
}

func (s *ThingDocument) updateDelta(state *thingDelta) bool {
	if s.Version > state.Version {
		// Received an old version; just ignore it.
		return false
	}
	s.Version = state.Version
	s.Timestamp = state.Timestamp
	s.State.Delta = state.State
	s.Metadata.Delta = state.Metadata
	return true
}

func updateStateRawCommon(state map[string]interface{}, update json.RawMessage) bool {
	if !hasUpdate(update) {
		return true
	}
	if update == nil {
		for k := range state {
			delete(state, k)
		}
		return true
	}
	return false
}

func updateStateRaw(state NestedState, update json.RawMessage) error {
	if updateStateRawCommon(state, update) {
		return nil
	}
	var u NestedState
	if err := json.Unmarshal([]byte(update), &u); err != nil {
		return ioterr.New(err, "unmarshaling update")
	}
	return updateState(state, u)
}

func updateStateMetadataRaw(state NestedMetadata, update json.RawMessage) error {
	if updateStateRawCommon(state, update) {
		return nil
	}
	var u NestedMetadata
	if err := json.Unmarshal([]byte(update), &u); err != nil {
		return ioterr.New(err, "unmarshaling update")
	}
	return updateState(state, u)
}

func updateState(state map[string]interface{}, update map[string]interface{}) error {
	if len(update) == 0 {
		for k := range state {
			delete(state, k)
		}
		return nil
	}
	for key, val := range update {
		switch v := val.(type) {
		case NestedState:
			if s, ok := state[key].(NestedState); ok {
				if err := updateState(s, v); err != nil {
					return ioterr.New(err, "updating state")
				}
			} else {
				state[key] = v
			}
		case nil:
			if _, ok := state[key]; ok {
				delete(state, key)
			}
		default:
			state[key] = v
		}
	}
	return nil
}

func hasUpdate(s json.RawMessage) bool {
	return len(s) != 0
}

// NestedState is JSON unmarshaller for state metadata.
type NestedState map[string]interface{}

func (n *NestedState) UnmarshalJSON(b []byte) error {
	if *n == nil {
		*n = make(map[string]interface{})
	}
	j := make(map[string]json.RawMessage)
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	for k, v := range j {
		v2, err := unmarshalStateImpl(v)
		if err != nil {
			return err
		}
		(*n)[k] = v2
	}
	return nil
}

func unmarshalStateImpl(b []byte) (interface{}, error) {
	switch {
	case b[0] == '[' && b[len(b)-1] == ']':
		var v2 []json.RawMessage
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		var v5 []interface{}
		for _, v3 := range v2 {
			v4, err := unmarshalStateImpl(v3)
			if err != nil {
				return nil, err
			}
			v5 = append(v5, v4)
		}
		return v5, nil
	case b[0] == '{':
		var v2 NestedState
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		return v2, nil
	default:
		var v2 interface{}
		if err := json.Unmarshal(b, &v2); err != nil {
			return nil, err
		}
		return v2, nil
	}
}

func (n *NestedState) MapTo(v interface{}) error {
	return nil
}
