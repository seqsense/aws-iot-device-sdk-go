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
	"reflect"
	"strings"
)

var errInvalidAttribute = errors.New("invalid attribute key")

func stateDiff(base, in interface{}) (interface{}, bool, error) {
	keys, hasChild, err := attributeKeys(base)
	if err != nil {
		return nil, false, err
	}
	if !hasChild {
		baseVal := reflect.ValueOf(base)
		inVal := reflect.ValueOf(in)
		if !baseVal.IsValid() {
			if !inVal.IsValid() {
				// Both nil
				return nil, false, nil
			}
			// One is nil
			return in, true, nil
		} else if !inVal.IsValid() {
			// One is nil
			return in, true, nil
		}
		switch baseVal.Type().Kind() {
		case reflect.Array, reflect.Slice:
			// Compare slice/array
			switch inVal.Type().Kind() {
			case reflect.Array, reflect.Slice:
				if baseVal.Len() != inVal.Len() {
					// Size differs
					return in, true, nil
				}
				for i := 0; i < baseVal.Len(); i++ {
					_, hasDiff, err := stateDiff(
						baseVal.Index(i).Interface(),
						inVal.Index(i).Interface(),
					)
					if err != nil {
						return nil, false, err
					}
					if hasDiff {
						return in, true, nil
					}
				}
			default:
				// Type differs
				return in, true, nil
			}
			return nil, false, nil
		default:
			// Compare primitive value
			if reflect.DeepEqual(base, in) {
				return nil, false, nil
			}
		}
		return in, true, nil
	}

	keysIn, hasChildIn, err := attributeKeys(in)
	if err != nil {
		return nil, false, err
	}
	if !hasChildIn {
		return in, true, nil
	}

	keysInMap := make(map[string]struct{})
	for _, k := range keysIn {
		keysInMap[k] = struct{}{}
	}
	baseMatcher, err := newAttributeMatcher(reflect.ValueOf(base))
	if err != nil {
		return nil, false, err
	}
	inMatcher, err := newAttributeMatcher(reflect.ValueOf(in))
	if err != nil {
		return nil, false, err
	}
	out := make(map[string]interface{})
	for _, k := range keys {
		if _, ok := keysInMap[k]; !ok {
			continue
		}
		delete(keysInMap, k)
		a, _ := baseMatcher.byKey(k)
		b, err := inMatcher.byKey(k)
		if err != nil {
			return nil, false, err
		}
		d, difer, err := stateDiff(a, b)
		if err != nil {
			return nil, false, err
		}
		if difer {
			out[k] = d
		}
	}
	for k := range keysInMap {
		b, _ := inMatcher.byKey(k)
		out[k] = b
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func attributeKeys(a interface{}) ([]string, bool, error) {
	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return nil, false, nil
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Map:
		out := make([]string, v.Len())
		keys := v.MapKeys()
		for i := range out {
			if keys[i].Kind() != reflect.String {
				return nil, false, ErrUnsupportedMapKeyType
			}
			out[i] = keys[i].String()
		}
		return out, true, nil
	case reflect.Struct:
		n := v.NumField()
		out := make([]string, 0, n)
		for i := 0; i < n; i++ {
			jsonName, ok := jsonFieldName(t.Field(i).Tag)
			if !ok {
				continue
			}
			if jsonName == "" {
				out = append(out, t.Field(i).Name)
			} else {
				out = append(out, jsonName)
			}
		}
		return out, true, nil
	case reflect.Ptr:
		return attributeKeys(v.Elem().Interface())
	}
	return nil, false, nil
}

type attributeMatcher struct {
	byName map[string]reflect.Value
}

func newAttributeMatcher(val reflect.Value) (*attributeMatcher, error) {
	if !val.IsValid() {
		return nil, errInvalidAttribute
	}
	t := val.Type()
	switch t.Kind() {
	case reflect.Struct:
		a := &attributeMatcher{byName: make(map[string]reflect.Value)}
		n := t.NumField()
		for i := 0; i < n; i++ {
			jsonName, ok := jsonFieldName(t.Field(i).Tag)
			if !ok {
				continue
			}
			var name string
			if jsonName == "" {
				name = t.Field(i).Name
			} else {
				name = jsonName
			}
			a.byName[name] = val.Field(i)
		}
		return a, nil
	case reflect.Map:
		a := &attributeMatcher{byName: make(map[string]reflect.Value)}
		for _, key := range val.MapKeys() {
			name := key.String()
			a.byName[name] = val.MapIndex(key)
		}
		return a, nil
	case reflect.Ptr, reflect.Interface:
		return newAttributeMatcher(val.Elem())
	}
	return nil, errInvalidAttribute
}

func (a *attributeMatcher) byKey(k string) (interface{}, error) {
	val, ok := a.byName[k]
	if !ok {
		return reflect.Value{}, errInvalidAttribute
	}
	return val.Interface(), nil
}

func jsonFieldName(t reflect.StructTag) (string, bool) {
	tag, ok := t.Lookup("json")
	if !ok {
		// Use struct field name.
		return "", true
	}
	if tag == "-" {
		// Field is ignored.
		return "", false
	}
	tags := strings.Split(tag, ",")
	if tags[0] == "" {
		// Use struct field name.
		return "", true
	}
	return tags[0], true
}
